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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Repo struct {
	Id         string
	Owner      string
	Name       string
	Repository *git.Repository
}

var mu sync.RWMutex
var repos map[string]Repo = make(map[string]Repo)

type TreeEntry struct {
	Name string            `json:"name"`
	Hash string            `json:"hash"`
	Mode filemode.FileMode `json:"mode"`
}

type TreeEntries struct {
	Entries []TreeEntry `json:"entries"`
}

type RepoInfo struct {
	Id    string `json:"id"`
	Owner string `json:"owner,omitempty"`
	Name  string `json:"name,omitempty"`
}

type RepoListResponse struct {
	Repos []RepoInfo `json:"repos"`
}

func AddFromPath(_ context.Context, id string, path string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, found := repos[id]; found {
		return fmt.Errorf("existing repo with id %s", id)
	}

	repository, err := git.PlainOpen(path)
	if err != nil {
		return err
	}

	repos[id] = Repo{
		Id:         id,
		Name:       "",
		Owner:      "",
		Repository: repository,
	}

	return nil
}

func AddFromGithub(_ context.Context, owner string, name string) error {
	mu.Lock()
	defer mu.Unlock()

	id := fmt.Sprintf("gh:%s:%s", owner, name)

	if _, found := repos[id]; found {
		return fmt.Errorf("existing repo with id %s", id)
	}

	url := fmt.Sprintf("https://github.com/%s/%s", owner, name)

	repoPath := filepath.Join(core.GetStoragePath(), id)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("creating repo directory %s: %w", repoPath, err)
	}

	repository, err := git.PlainClone(repoPath, true, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return err
	}

	repos[id] = Repo{
		Id:         id,
		Name:       name,
		Owner:      owner,
		Repository: repository,
	}

	return nil
}

func Get(_ context.Context, id string) (*git.Repository, error) {
	mu.RLock()
	defer mu.RUnlock()

	repo, found := repos[id]

	if found {
		return repo.Repository, nil
	} else {
		return nil, fmt.Errorf("no repo with id %s", id)
	}
}

func ResolveRepo(ctx context.Context, repoId string) (*git.Repository, error) {
	// Check if repo is already in memory
	if repo, err := Get(ctx, repoId); err == nil {
		return repo, nil
	}

	// Parse gh:owner:name format
	if !strings.HasPrefix(repoId, "gh:") {
		return nil, fmt.Errorf("repo id must be in gh:owner:name format, got: %s", repoId)
	}
	
	parts := strings.Split(repoId, ":")
	if len(parts) != 3 || parts[0] != "gh" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("repo id must be in gh:owner:name format, got: %s", repoId)
	}
	
	owner, name := parts[1], parts[2]
	
	// Check if repo exists in storage at gh/owner/name path
	repoPath := filepath.Join(core.GetStoragePath(), "gh", owner, name)
	if _, err := os.Stat(repoPath); err == nil {
		// Repo exists in storage, add it from path
		if err := AddFromPath(ctx, repoId, repoPath); err != nil {
			return nil, fmt.Errorf("failed to add repo from path %s: %w", repoPath, err)
		}
		return Get(ctx, repoId)
	}
	
	// Repo doesn't exist in storage, clone from GitHub
	if err := AddFromGithub(ctx, owner, name); err != nil {
		return nil, fmt.Errorf("failed to add repo from GitHub %s/%s: %w", owner, name, err)
	}
	
	return Get(ctx, repoId)
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
	response.Repos = make([]RepoInfo, 0, len(repos))

	for _, repo := range repos {
		response.Repos = append(response.Repos, RepoInfo{
			Id:    repo.Id,
			Name:  repo.Name,
			Owner: repo.Owner,
		})
	}

	sort.Slice(response.Repos, func(i, j int) bool {
		return response.Repos[i].Id < response.Repos[j].Id
	})

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
