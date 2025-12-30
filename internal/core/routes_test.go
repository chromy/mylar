package core

import (
	"fmt"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"testing"
	"time"
)

func TestIsValidMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{"GET method", http.MethodGet, true},
		{"HEAD method", http.MethodHead, true},
		{"POST method", http.MethodPost, true},
		{"PUT method", http.MethodPut, true},
		{"Invalid method INVALID", "INVALID", false},
		{"Invalid method CUSTOM", "CUSTOM", false},
		{"Empty method", "", false},
		{"Lowercase get", "get", false},
		{"Mixed case Post", "Post", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMethod(tt.method)
			if result != tt.expected {
				t.Errorf("isValidMethod(%q) = %v, want %v", tt.method, result, tt.expected)
			}
		})
	}
}

func TestRegisterValidMethod(t *testing.T) {
	route := Route{
		Id:      "test-get",
		Path:    "/test",
		Method:  http.MethodGet,
		Handler: func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {},
	}

	RegisterRoute(route)

	retrievedRoute, found := GetRoute("test-get")
	if !found {
		t.Error("Expected route to be registered")
	}
	if retrievedRoute.Method != http.MethodGet {
		t.Errorf("Expected method GET, got %s", retrievedRoute.Method)
	}
}

func TestRegisterInvalidMethodPanics(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"Invalid method INVALID", "INVALID"},
		{"Invalid method CUSTOM", "CUSTOM"},
		{"Empty method", ""},
		{"Lowercase get", "get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("RegisterRoute() should have panicked for invalid method %q", tt.method)
				}
			}()

			route := Route{
				Id:      "test-invalid-" + tt.method,
				Path:    "/test",
				Method:  tt.method,
				Handler: func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {},
			}

			RegisterRoute(route)
		})
	}
}

func TestRegisterAllValidMethods(t *testing.T) {
	validMethods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
	}

	for i, method := range validMethods {
		t.Run("RegisterRoute "+method, func(t *testing.T) {
			route := Route{
				Id:      fmt.Sprintf("test-%s-%d", method, i),
				Path:    "/test",
				Method:  method,
				Handler: func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {},
			}

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RegisterRoute() should not panic for valid method %q: %v", method, r)
				}
			}()

			RegisterRoute(route)
		})
	}
}

func createTestBlob(content string) *object.Blob {
	fs := memfs.New()
	repo, _ := git.Init(memory.NewStorage(), fs)
	worktree, _ := repo.Worktree()

	testFile, _ := fs.Create("test.txt")
	testFile.Write([]byte(content))
	testFile.Close()

	worktree.Add("test.txt")
	commitHash, _ := worktree.Commit("Test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})

	commit, _ := repo.CommitObject(commitHash)
	tree, _ := commit.Tree()
	entry, _ := tree.FindEntry("test.txt")
	obj, _ := repo.Object(plumbing.BlobObject, entry.Hash)
	return obj.(*object.Blob)
}

