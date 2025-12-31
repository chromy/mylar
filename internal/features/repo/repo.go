package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/viz/internal/core"
	"github.com/chromy/viz/internal/schemas"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"strings"
	"sync"
)

type State struct {
	Repos map[string]*git.Repository
}

var (
	mu    sync.RWMutex
	state State
)

type RepoInfo struct {
	Name string `json:"name"`
}

type TreeEntry struct {
	Name string          `json:"name"`
	Hash string          `json:"hash"`
	Mode filemode.FileMode `json:"mode"`
}

type RepoListResponse struct {
	Repos []RepoInfo `json:"repos"`
}

type TreeEntries struct {
	Entries []TreeEntry `json:"entries"`
}

func AddFromPath(_ context.Context, name string, path string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, found := state.Repos[name]; found {
		return fmt.Errorf("existing repo with name %s", name)
	}

	repository, err := git.PlainOpen(path)
	if err != nil {
		return err
	}

	state.Repos[name] = repository

	return nil
}

func Get(_ context.Context, name string) (*git.Repository, error) {
	mu.RLock()
	defer mu.RUnlock()

	repo, found := state.Repos[name]

	if found {
		return repo, nil
	} else {
		return nil, fmt.Errorf("no repo with name %s", name)
	}
}

func ResolveCommittishToHash(repo *git.Repository, committish string) (plumbing.Hash, error) {
	if plumbing.IsHash(committish) {
		hash := plumbing.NewHash(committish)
		return hash, nil
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(committish))
	if err == nil {
		return *hash, nil
	}

	return plumbing.ZeroHash, fmt.Errorf("unable to resolve committish '%s'", committish)
}

func ResolveCommittishToTreeish(repo *git.Repository, committish string) (plumbing.Hash, error) {
	hash, err := ResolveCommittishToHash(repo, committish)
	if err != nil {
		return hash, err
	}

	commit, err := repo.CommitObject(hash)
	if err == nil {
		return commit.TreeHash, nil
	}

	return hash, nil
}

func ListHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	mu.RLock()
	defer mu.RUnlock()

	response := RepoListResponse{}
	response.Repos = make([]RepoInfo, 0, len(state.Repos))

	for name, _ := range state.Repos {
		response.Repos = append(response.Repos, RepoInfo{
			Name: name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

var IsBinary = core.RegisterBlobComputation("isBinary", func(ctx context.Context, repoId string, hash plumbing.Hash) (bool, error) {
	repo, err := Get(ctx, repoId)
	if err != nil {
		return false, err
	}

	blob, err := repo.BlobObject(hash)
	if err != nil {
		return false, err
	}

	reader, err := blob.Reader()
	if err != nil {
		return false, err
	}
	defer reader.Close()

	buffer := make([]byte, 8000)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true, nil
		}
	}

	return false, nil
})

var Content = core.RegisterBlobComputation("content", func(ctx context.Context, repoId string, hash plumbing.Hash) (string, error) {

	isBinary, err := IsBinary(ctx, repoId, hash)
	if err != nil {
		return "", err
	}
	if isBinary {
		return "", nil
	}

	repo, err := Get(ctx, repoId)
	if err != nil {
		return "", err
	}

	blob, err := repo.BlobObject(hash)
	if err != nil {
		return "", err
	}

	reader, err := blob.Reader()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	buffer := make([]byte, blob.Size)
	_, err = reader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	return string(buffer), nil
})

var Lines = core.RegisterBlobComputation("lines", func(ctx context.Context, repoId string, hash plumbing.Hash) ([]string, error) {
	content, err := Content(ctx, repoId, hash)
	if err != nil {
		return nil, err
	}

	if content == "" {
		return []string{}, nil
	}

	lines := strings.Split(content, "\n")
	return lines, nil
})

var LineCount = core.RegisterBlobComputation("lineCount", func(ctx context.Context, repoId string, hash plumbing.Hash) (int64, error) {
	isBinary, err := IsBinary(ctx, repoId, hash)
	if err != nil {
		return 0, err
	}

	repo, err := Get(ctx, repoId)
	if err != nil {
		return 0, err
	}

	blob, err := repo.BlobObject(hash)
	if err != nil {
		return 0, err
	}

	reader, err := blob.Reader()
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	if isBinary {
		content, err := io.ReadAll(reader)
		if err != nil {
			return 0, err
		}
		return int64(len(content)), nil
	}

	lines, err := Lines(ctx, repoId, hash)
	if err != nil {
		return 0, err
	}

	return int64(len(lines)), nil
})

var GetTreeEntries = core.RegisterBlobComputation("treeEntries", func(ctx context.Context, repoId string, hash plumbing.Hash) ([]TreeEntry, error) {
	repo, err := Get(ctx, repoId)
	if err != nil {
		return nil, err
	}

	treeObj, err := repo.TreeObject(hash)
	if err != nil {
		return nil, err
	}

	var entries []TreeEntry
	for _, entry := range treeObj.Entries {
		entries = append(entries, TreeEntry{
			Name: entry.Name,
			Hash: entry.Hash.String(),
			Mode: entry.Mode,
		})
	}

	return entries, nil
})

var GetObjectType = core.RegisterBlobComputation("objectType", func(ctx context.Context, repoId string, hash plumbing.Hash) (string, error) {
	repo, err := Get(ctx, repoId)
	if err != nil {
		return "", err
	}

	obj, err := repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	if err != nil {
		return "", err
	}

	return obj.Type().String(), nil
})

func init() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Repos = make(map[string]*git.Repository)

	core.RegisterRoute(core.Route{
		Id:      "repo.list",
		Method:  http.MethodGet,
		Path:    "/api/repo",
		Handler: ListHandler,
	})

	schemas.Register("repo.RepoInfo", RepoInfo{})
	schemas.Register("repo.RepoListResponse", RepoListResponse{})
	schemas.Register("repo.TreeEntry", TreeEntry{})
	schemas.Register("repo.TreeEntries", TreeEntries{})
}
