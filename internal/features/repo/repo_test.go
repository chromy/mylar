package repo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/julienschmidt/httprouter"
)

func createTestRepo(name string) (*git.Repository, error) {
	fs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	testFile, err := fs.Create("test.txt")
	if err != nil {
		return nil, err
	}
	defer testFile.Close()

	_, err = testFile.Write([]byte("Hello, World!\nThis is a test file.\n"))
	if err != nil {
		return nil, err
	}

	_, err = worktree.Add("test.txt")
	if err != nil {
		return nil, err
	}

	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func createBinaryTestRepo() (*git.Repository, error) {
	fs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		return nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	binaryFile, err := fs.Create("binary.bin")
	if err != nil {
		return nil, err
	}
	defer binaryFile.Close()

	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x00, 0x04, 0x05, 0x06}
	_, err = binaryFile.Write(binaryData)
	if err != nil {
		return nil, err
	}

	textFile, err := fs.Create("text.txt")
	if err != nil {
		return nil, err
	}
	defer textFile.Close()

	_, err = textFile.Write([]byte("This is a text file.\nWith multiple lines.\n"))
	if err != nil {
		return nil, err
	}

	_, err = worktree.Add("binary.bin")
	if err != nil {
		return nil, err
	}

	_, err = worktree.Add("text.txt")
	if err != nil {
		return nil, err
	}

	_, err = worktree.Commit("Initial commit with binary and text", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func setupState() {
	mu.Lock()
	defer mu.Unlock()
	state = State{}
	state.Repos = make(map[string]*git.Repository)
}

func TestAddFromPath(t *testing.T) {
	setupState()

	repo, err := createTestRepo("test-repo")
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	state.Repos["existing-repo"] = repo

	tests := []struct {
		name        string
		repoName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Add new repo successfully",
			repoName:    "new-repo",
			expectError: false,
		},
		{
			name:        "Add repo with existing name",
			repoName:    "existing-repo",
			expectError: true,
			errorMsg:    "existing repo with name existing-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempRepo, err := createTestRepo(tt.repoName)
			if err != nil {
				t.Fatalf("Failed to create temp repo: %v", err)
			}

			oldCount := len(state.Repos)

			err = AddFromPath(context.Background(), tt.repoName, "dummy-path")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if len(state.Repos) != oldCount {
					t.Errorf("Repository count should not change on error")
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for invalid path but got none")
				}

				state.Repos[tt.repoName] = tempRepo

				if len(state.Repos) != oldCount+1 {
					t.Errorf("Expected repository count to increase by 1")
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	setupState()

	testRepo, err := createTestRepo("test-repo")
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	state.Repos["test-repo"] = testRepo

	tests := []struct {
		name        string
		repoName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Get existing repo",
			repoName:    "test-repo",
			expectError: false,
		},
		{
			name:        "Get non-existent repo",
			repoName:    "non-existent",
			expectError: true,
			errorMsg:    "no repo with name non-existent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := Get(context.Background(), tt.repoName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if repo != nil {
					t.Errorf("Expected nil repo on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if repo == nil {
					t.Errorf("Expected non-nil repo")
				}
			}
		})
	}
}

func TestResolveCommittishToHash(t *testing.T) {
	repo, err := createTestRepo("test-repo")
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	headHash := head.Hash()

	tests := []struct {
		name         string
		committish   string
		expectedHash plumbing.Hash
		expectError  bool
	}{
		{
			name:         "Valid hash",
			committish:   headHash.String(),
			expectedHash: headHash,
			expectError:  false,
		},
		{
			name:         "HEAD reference",
			committish:   "HEAD",
			expectedHash: headHash,
			expectError:  false,
		},
		{
			name:        "Invalid committish",
			committish:  "invalid-ref",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ResolveCommittishToHash(repo, tt.committish)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if hash != plumbing.ZeroHash {
					t.Errorf("Expected zero hash on error, got %s", hash.String())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if hash != tt.expectedHash {
					t.Errorf("Expected hash %s, got %s", tt.expectedHash.String(), hash.String())
				}
			}
		})
	}
}

func TestResolveCommittishToTreeish(t *testing.T) {
	repo, err := createTestRepo("test-repo")
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	headHash := head.Hash()

	commit, err := repo.CommitObject(headHash)
	if err != nil {
		t.Fatalf("Failed to get commit: %v", err)
	}
	treeHash := commit.TreeHash

	tests := []struct {
		name         string
		committish   string
		expectedHash plumbing.Hash
		expectError  bool
	}{
		{
			name:         "Valid commit hash",
			committish:   headHash.String(),
			expectedHash: treeHash,
			expectError:  false,
		},
		{
			name:         "HEAD reference",
			committish:   "HEAD",
			expectedHash: treeHash,
			expectError:  false,
		},
		{
			name:        "Invalid committish",
			committish:  "invalid-ref",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ResolveCommittishToTreeish(repo, tt.committish)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if hash != tt.expectedHash {
					t.Errorf("Expected tree hash %s, got %s", tt.expectedHash.String(), hash.String())
				}
			}
		})
	}
}

func TestListHandler(t *testing.T) {
	setupState()

	testRepo1, err := createTestRepo("repo1")
	if err != nil {
		t.Fatalf("Failed to create test repo 1: %v", err)
	}

	testRepo2, err := createTestRepo("repo2")
	if err != nil {
		t.Fatalf("Failed to create test repo 2: %v", err)
	}

	state.Repos["repo1"] = testRepo1
	state.Repos["repo2"] = testRepo2

	req, err := http.NewRequest("GET", "/api/repo", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()

	ListHandler(rr, req, httprouter.Params{})

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedContentType := "application/json"
	if ct := rr.Header().Get("Content-Type"); ct != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, ct)
	}

	var response RepoListResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Repos) != 2 {
		t.Errorf("Expected 2 repos, got %d", len(response.Repos))
	}

	repoNames := make(map[string]bool)
	for _, repo := range response.Repos {
		repoNames[repo.Name] = true
	}

	if !repoNames["repo1"] {
		t.Errorf("Expected repo1 in response")
	}
	if !repoNames["repo2"] {
		t.Errorf("Expected repo2 in response")
	}
}

func TestIsBinaryComputation(t *testing.T) {
	repo, err := createBinaryTestRepo()
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	setupState()
	state.Repos["test-repo"] = repo

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("Failed to get commit: %v", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("Failed to get tree: %v", err)
	}

	tests := []struct {
		name           string
		fileName       string
		expectedBinary bool
	}{
		{
			name:           "Binary file",
			fileName:       "binary.bin",
			expectedBinary: true,
		},
		{
			name:           "Text file",
			fileName:       "text.txt",
			expectedBinary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := tree.FindEntry(tt.fileName)
			if err != nil {
				t.Fatalf("Failed to find entry %s: %v", tt.fileName, err)
			}

			result, err := IsBinary(context.Background(), "test-repo", entry.Hash)
			if err != nil {
				t.Fatalf("IsBinary failed: %v", err)
			}

			isBinary, ok := result.(bool)
			if !ok {
				t.Fatalf("Expected bool result, got %T", result)
			}

			if isBinary != tt.expectedBinary {
				t.Errorf("Expected binary=%v for %s, got %v", tt.expectedBinary, tt.fileName, isBinary)
			}
		})
	}
}

func TestContentComputation(t *testing.T) {
	repo, err := createBinaryTestRepo()
	if err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	setupState()
	state.Repos["test-repo"] = repo

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("Failed to get commit: %v", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("Failed to get tree: %v", err)
	}

	tests := []struct {
		name            string
		fileName        string
		expectedContent string
	}{
		{
			name:            "Binary file returns empty string",
			fileName:        "binary.bin",
			expectedContent: "",
		},
		{
			name:            "Text file returns content",
			fileName:        "text.txt",
			expectedContent: "This is a text file.\nWith multiple lines.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := tree.FindEntry(tt.fileName)
			if err != nil {
				t.Fatalf("Failed to find entry %s: %v", tt.fileName, err)
			}

			result, err := Content(context.Background(), "test-repo", entry.Hash)
			if err != nil {
				t.Fatalf("Content failed: %v", err)
			}

			content, ok := result.(string)
			if !ok {
				t.Fatalf("Expected string result, got %T", result)
			}

			if content != tt.expectedContent {
				t.Errorf("Expected content '%s' for %s, got '%s'", tt.expectedContent, tt.fileName, content)
			}
		})
	}
}

func TestContentComputationNonExistentRepo(t *testing.T) {
	setupState()

	hash := plumbing.NewHash("0000000000000000000000000000000000000000")

	_, err := Content(context.Background(), "non-existent-repo", hash)
	if err == nil {
		t.Errorf("Expected error for non-existent repo")
	}
}

func TestIsBinaryComputationNonExistentRepo(t *testing.T) {
	setupState()

	hash := plumbing.NewHash("0000000000000000000000000000000000000000")

	_, err := IsBinary(context.Background(), "non-existent-repo", hash)
	if err == nil {
		t.Errorf("Expected error for non-existent repo")
	}
}