package api

import (
	"strings"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{"valid namespace", "my-namespace_1", false},
		{"empty namespace", "", true},
		{"invalid characters", "my namespace!", true},
		{"too long", strings.Repeat("a", 65), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNamespace(tt.namespace)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name      string
    key       string
		wantErr   bool
	}{
    {"valid key", "my-Key_1", false},
    {"empty key", "", true},
		{"invalid characters", "my namespace!", true},
    {"too long", strings.Repeat("a", 129), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKey(tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateValue(t *testing.T) {
	tests := []struct {
		name      string
    value     string
		wantErr   bool
	}{
    {"valid value", "my-Key_1", false},
    {"empty value", "", false},
    {"too long", strings.Repeat("a", 4097), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateValue(tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Root)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	var response map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&response)

	if response["error"] != "Page not found" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Info)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var response map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&response)

	if response["service"] != "config-service" {
		t.Errorf("expected service 'config-service', got '%s'", response["service"])
	}
	if response["version"] != "0.0.0" {
		t.Errorf("expected version '0.0.0', got '%s'", response["version"])
	}
}
