package core

import (
	"context"
	"github.com/go-git/go-git/v5/plumbing"
//	"github.com/go-git/go-git/v5/plumbing/object"
//	"fmt"
	"testing"
)

func TestGetBlobComputation(t *testing.T) {
	RegisterBlobComputation("TestGetBlobComputation",  func(ctx context.Context, hash plumbing.Hash) (interface{}, error) {
			return "result", nil
	})

	c, found := GetBlobComputation("TestGetBlobComputation")

	if !found {
		t.Error("Expected blob computation to be registered")
	}

	if c.Id != "TestGetBlobComputation" {
		t.Errorf("Expected id 'TestGetBlobComputation', got %s", c.Id)
	}
}

func TestExecuteBlobComputation(t *testing.T) {
	RegisterBlobComputation("TestExecuteBlobComputation",  func(ctx context.Context, hash plumbing.Hash) (interface{}, error) {
			return "result", nil
	})

	c, _ := GetBlobComputation("TestExecuteBlobComputation")

	result, err := c.Execute(context.Background(), plumbing.NewHash("efc4fcc2e78479e60133c9dcb3460c45a1c0efa9"))

	if err != nil {
		t.Error("Expected blob computation to succeed")
	}

	if result != "result" {
		t.Errorf("Expected result to be %v got %v", "result", result)
	}
}

func TestBlobComputationCachesResults(t *testing.T) {
	callCount := 0

	f := RegisterBlobComputation("test2",  func(ctx context.Context, hash plumbing.Hash) (interface{}, error) {
		callCount += 1
		return hash.String(), nil
	})

	hash := plumbing.NewHash("efc4fcc2e78479e60133c9dcb3460c45a1c0efa9")

	a, aErr := f(context.Background(), hash)
	b, bErr := f(context.Background(), hash)

	if aErr != nil {
		t.Error("Expected a computation to succeed")
	}
	if bErr != nil {
		t.Error("Expected b computation to succeed")
	}
	if a != hash.String() {
		t.Errorf("Expected a to be %v got %v", hash.String(), a)
	}
	if b != hash.String() {
		t.Errorf("Expected b to be %v got %v", hash.String(), b)
	}

	if callCount != 1 {
		t.Errorf("Expected single call")
	}
}
