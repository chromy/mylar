package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromy/viz/internal/routes"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path/filepath"
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

type RepoListResponse struct {
	Repos []RepoInfo `json:"repos"`
}

type FileSystemEntry struct {
	Name     string            `json:"name"`
	Path     string            `json:"path"`
	Type     string            `json:"type"`               // "file" or "directory"
	Size     *int64            `json:"size,omitempty"`     // Only for files
	Hash     string            `json:"hash,omitempty"`     // Only for files
	Children []FileSystemEntry `json:"children,omitempty"` // Only for directories
}

type InfoResponse struct {
	Entry FileSystemEntry `json:"entry"`
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

func ResolveCommitish(repo *git.Repository, commitish string) (plumbing.Hash, error) {
	if commitish == "" {
		ref, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("failed to get HEAD: %w", err)
		}
		return ref.Hash(), nil
	}

	hash := plumbing.NewHash(commitish)
	if !hash.IsZero() {
		_, err := repo.CommitObject(hash)
		if err == nil {
			return hash, nil
		}
	}

	ref, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+commitish), true)
	if err == nil {
		return ref.Hash(), nil
	}

	ref, err = repo.Reference(plumbing.ReferenceName("refs/tags/"+commitish), true)
	if err == nil {
		return ref.Hash(), nil
	}

	ref, err = repo.Reference(plumbing.ReferenceName("refs/remotes/origin/"+commitish), true)
	if err == nil {
		return ref.Hash(), nil
	}

	return plumbing.ZeroHash, fmt.Errorf("unable to resolve commitish '%s'", commitish)
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

func RawHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	repoName := ps.ByName("repo")
	filePath := ps.ByName("path")

	repo, err := Get(r.Context(), repoName)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	ref, err := repo.Head()
	if err != nil {
		http.Error(w, "Failed to get repository HEAD", http.StatusInternalServerError)
		return
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		http.Error(w, "Failed to get commit", http.StatusInternalServerError)
		return
	}

	tree, err := commit.Tree()
	if err != nil {
		http.Error(w, "Failed to get tree", http.StatusInternalServerError)
		return
	}

	file, err := tree.File(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	contents, err := file.Contents()
	if err != nil {
		http.Error(w, "Failed to read file contents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(contents))
}

func InfoHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	mu.RLock()
	defer mu.RUnlock()

	repoName := ps.ByName("repo")
	targetPath := ps.ByName("path")

	repo, found := state.Repos[repoName]
	if !found {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	ref, err := repo.Head()
	if err != nil {
		http.Error(w, "Failed to get repository HEAD", http.StatusInternalServerError)
		return
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		http.Error(w, "Failed to get commit", http.StatusInternalServerError)
		return
	}

	tree, err := commit.Tree()
	if err != nil {
		http.Error(w, "Failed to get tree", http.StatusInternalServerError)
		return
	}

	file, fileErr := tree.File(targetPath)
	if fileErr == nil {
		size := file.Size
		response := InfoResponse{
			Entry: FileSystemEntry{
				Name: filepath.Base(targetPath),
				Path: targetPath,
				Type: "file",
				Size: &size,
				Hash: file.Hash.String(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
		return
	}

	var dirTree *object.Tree
	if targetPath == "" || targetPath == "/" {
		dirTree = tree
	} else {
		dirEntry, err := tree.FindEntry(targetPath)
		if err != nil {
			http.Error(w, "Path not found", http.StatusNotFound)
			return
		}

		if dirEntry.Mode != filemode.Dir {
			http.Error(w, "Path not found", http.StatusNotFound)
			return
		}

		dirTree, err = tree.Tree(targetPath)
		if err != nil {
			http.Error(w, "Directory not found", http.StatusNotFound)
			return
		}
	}

	// Build directory response with children
	children := make([]FileSystemEntry, 0)
	for _, entry := range dirTree.Entries {
		childPath := targetPath
		if childPath != "" && !strings.HasSuffix(childPath, "/") {
			childPath += "/"
		}
		childPath += entry.Name

		if entry.Mode == filemode.Dir {
			children = append(children, FileSystemEntry{
				Name: entry.Name,
				Path: childPath,
				Type: "directory",
				Hash: entry.Hash.String(),
			})
		} else {
			// For files, we need to get the file object to get the size
			childFile, err := dirTree.File(entry.Name)
			if err == nil {
				size := childFile.Size
				children = append(children, FileSystemEntry{
					Name: entry.Name,
					Path: childPath,
					Type: "file",
					Size: &size,
					Hash: entry.Hash.String(),
				})
			}
		}
	}

	response := InfoResponse{
		Entry: FileSystemEntry{
			Name:     filepath.Base(targetPath),
			Path:     targetPath,
			Type:     "directory",
			Hash:     dirTree.Hash.String(),
			Children: children,
		},
	}

	if targetPath == "" || targetPath == "/" {
		response.Entry.Name = "/"
		response.Entry.Path = ""
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func init() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Repos = make(map[string]*git.Repository)

	routes.Register(routes.Route{
		Id:      "repo.list",
		Method:  http.MethodGet,
		Path:    "/api/repo",
		Handler: ListHandler,
	})

	routes.Register(routes.Route{
		Id:      "repo.raw",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/raw/*path",
		Handler: RawHandler,
	})

	routes.Register(routes.Route{
		Id:      "repo.info",
		Method:  http.MethodGet,
		Path:    "/api/repo/:repo/info/*path",
		Handler: InfoHandler,
	})

}
