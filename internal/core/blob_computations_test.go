package core

import (
	"context"
	"github.com/go-git/go-git/v5/plumbing"
	//	"github.com/go-git/go-git/v5/plumbing/object"
	//	"fmt"
	"testing"
)

func TestGetBlobComputation(t *testing.T) {
	RegisterBlobComputation("TestGetBlobComputation", func(ctx context.Context, _ string, hash plumbing.Hash) (string, error) {
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
	RegisterBlobComputation("TestExecuteBlobComputation", func(ctx context.Context, _ string, hash plumbing.Hash) (string, error) {
		return "result", nil
	})

	c, _ := GetBlobComputation("TestExecuteBlobComputation")

	result, err := c.Execute(context.Background(), "", plumbing.NewHash("efc4fcc2e78479e60133c9dcb3460c45a1c0efa9"))

	if err != nil {
		t.Error("Expected blob computation to succeed")
	}

	if result != "result" {
		t.Errorf("Expected result to be %v got %v", "result", result)
	}
}

func TestBlobComputationCachesResults(t *testing.T) {
	callCount := 0

	f := RegisterBlobComputation("TestBlobComputationCachesResults", func(ctx context.Context, _ string, hash plumbing.Hash) (string, error) {
		callCount += 1
		return hash.String(), nil
	})

	hash := plumbing.NewHash("efc4fcc2e78479e60133c9dcb3460c45a1c0efa9")

	a, aErr := f(context.Background(), "", hash)
	b, bErr := f(context.Background(), "", hash)

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

type CustomStruct struct {
	Field1 string
	Field2 int
	Field3 []string
}

func TestBlobComputationPreservesTypeAfterCaching(t *testing.T) {
	ResetBlobComputationsForTesting()
	callCount := 0

	f := RegisterBlobComputation("TestBlobComputationPreservesType", func(ctx context.Context, _ string, hash plumbing.Hash) (CustomStruct, error) {
		callCount += 1
		return CustomStruct{
			Field1: "test",
			Field2: 42,
			Field3: []string{"a", "b", "c"},
		}, nil
	})

	hash := plumbing.NewHash("abc4fcc2e78479e60133c9dcb3460c45a1c0efa9")

	// First call - direct from function
	firstResult, err := f(context.Background(), "", hash)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	// Second call - should come from cache
	secondResult, err := f(context.Background(), "", hash)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	// Verify only called once
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d calls", callCount)
	}

	// Check that first result has correct type and values
	if firstResult.Field1 != "test" || firstResult.Field2 != 42 || len(firstResult.Field3) != 3 {
		t.Errorf("First result has incorrect values: %+v", firstResult)
	}

	// Check that cached result also has correct type and values
	if secondResult.Field1 != "test" || secondResult.Field2 != 42 || len(secondResult.Field3) != 3 {
		t.Errorf("Second result has incorrect values: %+v", secondResult)
	}
}

func TestBlobComputationPreservesSliceTypeAfterCaching(t *testing.T) {
	ResetBlobComputationsForTesting()
	callCount := 0

	f := RegisterBlobComputation("TestBlobComputationPreservesSliceType", func(ctx context.Context, _ string, hash plumbing.Hash) ([]string, error) {
		callCount += 1
		return []string{"line1", "line2", "line3"}, nil
	})

	hash := plumbing.NewHash("def4fcc2e78479e60133c9dcb3460c45a1c0efa9")

	// First call - direct from function
	firstResult, err := f(context.Background(), "", hash)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	// Second call - should come from cache
	secondResult, err := f(context.Background(), "", hash)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	// Verify only called once
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d calls", callCount)
	}

	// Check that first result has correct type and values
	if len(firstResult) != 3 || firstResult[0] != "line1" {
		t.Errorf("First result has incorrect values: %+v", firstResult)
	}

	// Check that cached result also has correct type and values
	if len(secondResult) != 3 || secondResult[0] != "line1" {
		t.Errorf("Second result has incorrect values: %+v", secondResult)
	}
}
