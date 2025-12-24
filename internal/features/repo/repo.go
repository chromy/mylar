package repo;

import (
	"github.com/go-git/go-git/v5"
	"sync"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"fmt"
	"github.com/chromy/viz/internal/routes"
	"encoding/json"
)

type State struct {
	Repos map[string]*git.Repository
}

var (
	mu  sync.RWMutex
	state State
)

type RepoInfo struct {
	Name string `json:"name"`
}

type RepoListResponse struct {
	Repos []RepoInfo `json:"repos"`
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
	mu.RLock()
	defer mu.RUnlock()

	repoName := ps.ByName("repo")
	path := ps.ByName("path")

	fmt.Fprintf(w, "repo: %s path: %s\n", repoName, path)
}

func AddFromPath(name string, path string) error {
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

func init() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Repos = make(map[string]*git.Repository)

	routes.Register(routes.Route{
		Id: "repo.list",
		Method: http.MethodGet,
		Path: "/api/repo",
		Handler: ListHandler,
	})

	routes.Register(routes.Route{
		Id: "repo.raw",
		Method: http.MethodGet,
		Path: "/api/repo/:repo/raw/*path",
		Handler: RawHandler,
	})
}
