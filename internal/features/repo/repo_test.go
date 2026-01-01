package repo

import (
	"context"
	"testing"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/chromy/viz/internal/core"
)

func TestGetCommitRootTreeHash(t *testing.T) {
	// Test with a non-existent repo
	_, err := GetCommitRootTreeHash(context.Background(), "nonexistent", plumbing.NewHash("0123456789abcdef0123456789abcdef01234567"))
	if err == nil {
		t.Error("Expected error for non-existent repo")
	}
}

func TestGetCommitRootTreeHashRegistration(t *testing.T) {
	// Test that the blob computation was registered
	computation, found := core.GetBlobComputation("commitRootTreeHash")
	if !found {
		t.Error("Expected commitRootTreeHash blob computation to be registered")
	}
	
	if computation.Id != "commitRootTreeHash" {
		t.Errorf("Expected id 'commitRootTreeHash', got %s", computation.Id)
	}
}

