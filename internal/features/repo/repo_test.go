package repo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/julienschmidt/httprouter"
)

func TestRawHandler(t *testing.T) {
	fs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	testContent := "Hello, World!\nThis is a test file."

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	testFile, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = testFile.Write([]byte(testContent))
	if err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	testFile.Close()

	_, err = worktree.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitHash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	ref := plumbing.NewHashReference(plumbing.HEAD, commitHash)
	err = repo.Storer.SetReference(ref)
	if err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	mu.Lock()
	originalRepos := state.Repos
	if state.Repos == nil {
		state.Repos = make(map[string]*git.Repository)
	}
	state.Repos["test-repo"] = repo
	mu.Unlock()

	defer func() {
		mu.Lock()
		state.Repos = originalRepos
		mu.Unlock()
	}()

	tests := []struct {
		name           string
		repoName       string
		filePath       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid file request",
			repoName:       "test-repo",
			filePath:       "test.txt",
			expectedStatus: http.StatusOK,
			expectedBody:   testContent,
		},
		{
			name:           "Repository not found",
			repoName:       "nonexistent-repo",
			filePath:       "test.txt",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Repository not found\n",
		},
		{
			name:           "File not found",
			repoName:       "test-repo",
			filePath:       "nonexistent.txt",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "File not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/repo/"+tt.repoName+"/raw/"+tt.filePath, nil)
			w := httptest.NewRecorder()

			params := httprouter.Params{
				{Key: "repo", Value: tt.repoName},
				{Key: "path", Value: tt.filePath},
			}

			RawHandler(w, req, params)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestInfoHandler(t *testing.T) {
	fs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	testContent := "Hello, World!\nThis is a test file."

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	testFile, err := fs.Create("test.txt")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = testFile.Write([]byte(testContent))
	if err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	testFile.Close()

	_, err = worktree.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commitHash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	ref := plumbing.NewHashReference(plumbing.HEAD, commitHash)
	err = repo.Storer.SetReference(ref)
	if err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	mu.Lock()
	originalRepos := state.Repos
	if state.Repos == nil {
		state.Repos = make(map[string]*git.Repository)
	}
	state.Repos["test-repo"] = repo
	mu.Unlock()

	defer func() {
		mu.Lock()
		state.Repos = originalRepos
		mu.Unlock()
	}()

	tests := []struct {
		name           string
		repoName       string
		filePath       string
		expectedStatus int
		expectJSON     bool
	}{
		{
			name:           "Valid file request",
			repoName:       "test-repo",
			filePath:       "test.txt",
			expectedStatus: http.StatusOK,
			expectJSON:     true,
		},
		{
			name:           "Repository not found",
			repoName:       "nonexistent-repo",
			filePath:       "test.txt",
			expectedStatus: http.StatusNotFound,
			expectJSON:     false,
		},
		{
			name:           "File not found",
			repoName:       "test-repo",
			filePath:       "nonexistent.txt",
			expectedStatus: http.StatusNotFound,
			expectJSON:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/repo/"+tt.repoName+"/info/"+tt.filePath, nil)
			w := httptest.NewRecorder()

			params := httprouter.Params{
				{Key: "repo", Value: tt.repoName},
				{Key: "path", Value: tt.filePath},
			}

			InfoHandler(w, req, params)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectJSON {
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
				}

				var response InfoResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON response: %v", err)
				}

				if response.Entry.Name != "test.txt" {
					t.Errorf("Expected file name test.txt, got %s", response.Entry.Name)
				}

				if response.Entry.Path != tt.filePath {
					t.Errorf("Expected file path %s, got %s", tt.filePath, response.Entry.Path)
				}

				if response.Entry.Type != "file" {
					t.Errorf("Expected type file, got %s", response.Entry.Type)
				}

				if response.Entry.Size == nil || *response.Entry.Size != int64(len(testContent)) {
					t.Errorf("Expected file size %d, got %v", len(testContent), response.Entry.Size)
				}

				if response.Entry.Hash == "" {
					t.Error("Expected non-empty hash")
				}
			}
		})
	}
}

func TestInfoHandlerDirectories(t *testing.T) {
	fs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Create directory structure
	// /root.txt
	// /subdir/nested.txt
	// /subdir/another.txt

	rootFile, err := fs.Create("root.txt")
	if err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}
	rootFile.Write([]byte("root content"))
	rootFile.Close()

	err = fs.MkdirAll("subdir", 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	nestedFile, err := fs.Create("subdir/nested.txt")
	if err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}
	nestedFile.Write([]byte("nested content"))
	nestedFile.Close()

	anotherFile, err := fs.Create("subdir/another.txt")
	if err != nil {
		t.Fatalf("Failed to create another file: %v", err)
	}
	anotherFile.Write([]byte("another content"))
	anotherFile.Close()

	// Add all files to git
	_, err = worktree.Add("root.txt")
	if err != nil {
		t.Fatalf("Failed to add root.txt: %v", err)
	}

	_, err = worktree.Add("subdir/nested.txt")
	if err != nil {
		t.Fatalf("Failed to add nested.txt: %v", err)
	}

	_, err = worktree.Add("subdir/another.txt")
	if err != nil {
		t.Fatalf("Failed to add another.txt: %v", err)
	}

	commitHash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	ref := plumbing.NewHashReference(plumbing.HEAD, commitHash)
	err = repo.Storer.SetReference(ref)
	if err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	mu.Lock()
	originalRepos := state.Repos
	if state.Repos == nil {
		state.Repos = make(map[string]*git.Repository)
	}
	state.Repos["test-repo"] = repo
	mu.Unlock()

	defer func() {
		mu.Lock()
		state.Repos = originalRepos
		mu.Unlock()
	}()

	tests := []struct {
		name             string
		path             string
		expectedType     string
		expectedChildren int
	}{
		{
			name:             "Root directory",
			path:             "",
			expectedType:     "directory",
			expectedChildren: 2, // root.txt and subdir
		},
		{
			name:             "Subdirectory",
			path:             "subdir",
			expectedType:     "directory",
			expectedChildren: 2, // nested.txt and another.txt
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/repo/test-repo/info/"+tt.path, nil)
			w := httptest.NewRecorder()

			params := httprouter.Params{
				{Key: "repo", Value: "test-repo"},
				{Key: "path", Value: tt.path},
			}

			InfoHandler(w, req, params)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			var response InfoResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to unmarshal JSON response: %v", err)
			}

			if response.Entry.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, response.Entry.Type)
			}

			if len(response.Entry.Children) != tt.expectedChildren {
				t.Errorf("Expected %d children, got %d", tt.expectedChildren, len(response.Entry.Children))
			}

			if response.Entry.Hash == "" {
				t.Error("Expected non-empty hash for directory")
			}

			// Verify children have correct types
			for _, child := range response.Entry.Children {
				if child.Type == "file" {
					if child.Size == nil {
						t.Errorf("Expected size for file %s", child.Name)
					}
				} else if child.Type == "directory" {
					if child.Size != nil {
						t.Errorf("Expected no size for directory %s", child.Name)
					}
				}
			}
		})
	}
}
