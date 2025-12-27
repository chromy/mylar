package schemas

import (
	"reflect"
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

	RegisterStruct("test-struct", testStruct)

	schema, found := Get("test-struct")
	if !found {
		t.Error("Expected schema to be registered")
	}
	if schema.Id != "test-struct" {
		t.Errorf("Expected schema ID 'test-struct', got %s", schema.Id)
	}
	if schema.Type != reflect.TypeOf(testStruct) {
		t.Errorf("Expected schema type %v, got %v", reflect.TypeOf(testStruct), schema.Type)
	}
	if schema.Schema == "" {
		t.Error("Expected schema to contain Zod schema string")
	}
}

func TestRegisterDuplicateIdPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("RegisterStruct() should have panicked for duplicate ID")
		}
	}()

	testStruct := TestStruct{Name: "test", Age: 25}
	RegisterStruct("duplicate-test", testStruct)
	RegisterStruct("duplicate-test", testStruct)
}

func TestGetNonExistentSchema(t *testing.T) {
	_, found := Get("non-existent")
	if found {
		t.Error("Expected schema not to be found")
	}
}

func TestListSchemas(t *testing.T) {
	RegisterStruct("list-test-1", TestStruct{})
	RegisterStruct("list-test-2", TestStruct{})

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

func TestGetAllSchemas(t *testing.T) {
	RegisterStruct("get-all-test", TestStruct{})

	allSchemas := GetAllSchemas()
	if len(allSchemas) == 0 {
		t.Error("Expected at least one schema")
	}

	schema, found := allSchemas["get-all-test"]
	if !found {
		t.Error("Expected 'get-all-test' schema to be in GetAllSchemas result")
	}
	if schema == "" {
		t.Error("Expected schema to contain Zod schema string")
	}
}