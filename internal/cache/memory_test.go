package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_AddAndGet(t *testing.T) {
	cache := NewMemoryCache()

	key := "test_key"
	value := []byte("test_value")

	// Add item to cache
	err := cache.Add(key, value)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Retrieve item from cache
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Compare values
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}
}

func TestMemoryCache_GetNonExistent(t *testing.T) {
	cache := NewMemoryCache()

	_, err := cache.Get("non_existent_key")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryCache_NoExpiration(t *testing.T) {
	cache := NewMemoryCache()

	key := "persistent_key"
	value := []byte("persistent_value")

	// Add item to cache
	err := cache.Add(key, value)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Item should exist immediately
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}

	// Wait some time
	time.Sleep(50 * time.Millisecond)

	// Item should still exist (no expiration)
	retrieved, err = cache.Get(key)
	if err != nil {
		t.Errorf("Expected item to persist, got error: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}
}

func TestMemoryCache_DataIsolation(t *testing.T) {
	cache := NewMemoryCache()

	key := "isolation_test"
	originalValue := []byte("original")

	// Add item to cache
	err := cache.Add(key, originalValue)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Modify the original slice
	originalValue[0] = 'X'

	// Retrieved value should be unchanged
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved[0] == 'X' {
		t.Error("Cache value was modified by external change to original slice")
	}

	// Modify retrieved slice
	retrieved[0] = 'Y'

	// Cache should still have original value
	retrieved2, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved2[0] == 'Y' {
		t.Error("Cache value was modified by external change to retrieved slice")
	}
}

func TestMemoryCache_Size(t *testing.T) {
	cache := NewMemoryCache()

	if cache.Size() != 0 {
		t.Errorf("Expected empty cache to have size 0, got %d", cache.Size())
	}

	// Add items
	cache.Add("key1", []byte("value1"))
	cache.Add("key2", []byte("value2"))

	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}

	// Add another item
	cache.Add("key3", []byte("value3"))

	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache()

	// Add items
	cache.Add("key1", []byte("value1"))
	cache.Add("key2", []byte("value2"))

	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected empty cache after clear, got size %d", cache.Size())
	}

	// Verify items are gone
	_, err := cache.Get("key1")
	if err != ErrNotFound {
		t.Error("Expected key1 to be not found after clear")
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache()

	// This is a basic concurrent access test
	// In a more comprehensive test suite, you might want to use
	// more sophisticated race condition testing
	done := make(chan bool, 2)

	// Goroutine 1: Add items
	go func() {
		for i := 0; i < 100; i++ {
			key := "key" + string(rune('0'+i%10))
			value := []byte("value" + string(rune('0'+i%10)))
			cache.Add(key, value)
		}
		done <- true
	}()

	// Goroutine 2: Read items
	go func() {
		for i := 0; i < 100; i++ {
			key := "key" + string(rune('0'+i%10))
			cache.Get(key) // Don't care about the result, just testing for races
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Test passed if we get here without race conditions
}
