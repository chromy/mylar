package routes

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"testing"
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

	Register(route)

	retrievedRoute, found := Get("test-get")
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
					t.Errorf("Register() should have panicked for invalid method %q", tt.method)
				}
			}()

			route := Route{
				Id:      "test-invalid-" + tt.method,
				Path:    "/test",
				Method:  tt.method,
				Handler: func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {},
			}

			Register(route)
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
		t.Run("Register "+method, func(t *testing.T) {
			route := Route{
				Id:      fmt.Sprintf("test-%s-%d", method, i),
				Path:    "/test",
				Method:  method,
				Handler: func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {},
			}

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Register() should not panic for valid method %q: %v", method, r)
				}
			}()

			Register(route)
		})
	}
}
