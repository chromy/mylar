package schemas

import (
	"testing"
)

type TestStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestRegisterStruct(t *testing.T) {
	testStruct := TestStruct{
		Name: "test",
		Age:  25,
	}

	Register("test-struct", testStruct)

	schema, found := Get("test-struct")
	if !found {
		t.Error("Expected schema to be registered")
	}
	if schema.Id != "test-struct" {
		t.Errorf("Expected schema ID 'test-struct', got %s", schema.Id)
	}
}

func TestRegisterDuplicateIdPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("RegisterStruct() should have panicked for duplicate ID")
		}
	}()

	testStruct := TestStruct{Name: "test", Age: 25}
	Register("duplicate-test", testStruct)
	Register("duplicate-test", testStruct)
}

func TestGetNonExistentSchema(t *testing.T) {
	_, found := Get("non-existent")
	if found {
		t.Error("Expected schema not to be found")
	}
}

func TestListSchemas(t *testing.T) {
	Register("list-test-1", TestStruct{})
	Register("list-test-2", TestStruct{})

	list := List()
	if len(list) < 2 {
		t.Errorf("Expected at least 2 schemas, got %d", len(list))
	}

	// Check that both test schemas are in the list
	found1, found2 := false, false
	for _, id := range list {
		if id == "list-test-1" {
			found1 = true
		}
		if id == "list-test-2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("Expected both test schemas to be in the list")
	}
}
