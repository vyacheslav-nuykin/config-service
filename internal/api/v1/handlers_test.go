package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vyacheslav-nuykin/config-service/internal/storage"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/config_service?sslmode=disable"
	}

	ctx := context.Background()

	if err := storage.RunMigrations(dbURL); err != nil {
		panic("failed to run migrations: " + err.Error())
	}

	var err error
	testPool, err = storage.New(ctx, dbURL)
	if err != nil {
		panic("failed to connect to test db: " + err.Error())
	}

	code := m.Run()

	testPool.Close()
	os.Exit(code)
}

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
		errField  string
		errReason string
	}{
		{"valid namespace", "my-namespace_1", false, "", ""},
		{"empty namespace", "", true, "namespace", "cannot be empty"},
		{"invalid characters", "my namespace!", true, "namespace", "contains invalid characters"},
		{"too long", strings.Repeat("a", 65), true, "namespace", "length must be between 1 and 64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNamespace(tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if valErr.Field != tt.errField {
					t.Errorf("expected field %q, got %q", tt.errField, valErr.Field)
				}
				if valErr.Reason != tt.errReason {
					t.Errorf("expected reason %q, got %q", tt.errReason, valErr.Reason)
				}
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantErr   bool
		errField  string
		errReason string
	}{
		{"valid key", "my-Key_1", false, "", ""},
		{"empty key", "", true, "key", "cannot be empty"},
		{"invalid characters", "my namespace!", true, "key", "contains invalid characters"},
		{"too long", strings.Repeat("a", 129), true, "key", "length must be between 1 and 128"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if valErr.Field != tt.errField {
					t.Errorf("expected field %q, got %q", tt.errField, valErr.Field)
				}
				if valErr.Reason != tt.errReason {
					t.Errorf("expected reason %q, got %q", tt.errReason, valErr.Reason)
				}
			}
		})
	}
}

func TestValidateValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantErr   bool
		errField  string
		errReason string
	}{
		{"valid value", "my-Key_1", false, "", ""},
		{"empty value", "", false, "", ""},
		{"too long", strings.Repeat("a", 4097), true, "value", "length must be between 0 and 4096"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if valErr.Field != tt.errField {
					t.Errorf("expected field %q, got %q", tt.errField, valErr.Field)
				}
				if valErr.Reason != tt.errReason {
					t.Errorf("expected reason %q, got %q", tt.errReason, valErr.Reason)
				}
			}
		})
	}
}

func TestWriteValidationError(t *testing.T) {
	t.Run("ValidationError", func(t *testing.T) {
		rr := httptest.NewRecorder()
		valErr := &ValidationError{ErrorStr: "validation failed", Field: "namespace", Reason: "cannot be empty"}
		writeValidationError(rr, valErr)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
		var resp ValidationError
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp != *valErr {
			t.Errorf("expected %+v, got %+v", valErr, resp)
		}
	})

	t.Run("generic error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		genericErr := errors.New("some error")
		writeValidationError(rr, genericErr)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
		var resp map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["error"] != "some error" {
			t.Errorf("expected error message, got %q", resp["error"])
		}
	})
}

func TestRoot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Root)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var response map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&response)
	if response["error"] != "Page not found" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rr := httptest.NewRecorder()
	Info(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var response map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&response)
	if response["service"] != "config-service" {
		t.Errorf("expected service 'config-service', got '%s'", response["service"])
	}
	if response["version"] != "0.0.0" {
		t.Errorf("expected version '0.0.0', got '%s'", response["version"])
	}

	os.Setenv("SERVICE", "my-service")
	os.Setenv("VERSION", "1.2.3")
	defer os.Unsetenv("SERVICE")
	defer os.Unsetenv("VERSION")

	req = httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rr = httptest.NewRecorder()
	Info(rr, req)

	_ = json.NewDecoder(rr.Body).Decode(&response)
	if response["service"] != "my-service" {
		t.Errorf("expected service 'my-service', got '%s'", response["service"])
	}
	if response["version"] != "1.2.3" {
		t.Errorf("expected version '1.2.3', got '%s'", response["version"])
	}
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	handler := Health(testPool)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" || resp["database"] != "ok" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestSetConfig(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		key        string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "valid config",
			namespace:  "test-ns",
			key:        "test-key",
			body:       `{"value": "test-value"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid namespace",
			namespace:  "invalid namespace!",
			key:        "test-key",
			body:       `{"value": "test-value"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
		{
			name:       "invalid key",
			namespace:  "test-ns",
			key:        "invalid key!",
			body:       `{"value": "test-value"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
		{
			name:       "value too long",
			namespace:  "test-ns",
			key:        "test-key",
			body:       fmt.Sprintf(`{"value": "%s"}`, strings.Repeat("a", 4097)),
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
		{
			name:       "invalid JSON",
			namespace:  "test-ns",
			key:        "test-key",
			body:       `{"value": "test-value"`,
			wantStatus: http.StatusBadRequest,
			wantError:  "unexpected EOF",
		},
		{
			name:       "body too large",
			namespace:  "test-ns",
			key:        "test-key",
			body:       fmt.Sprintf(`{"value": "%s"}`, strings.Repeat("a", 8193)),
			wantStatus: http.StatusBadRequest,
			wantError:  "http: request body too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = storage.DeleteConfig(context.Background(), testPool, tt.namespace, tt.key)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/config/x/x", bytes.NewBufferString(tt.body))
			req.SetPathValue("namespace", tt.namespace)
			req.SetPathValue("key", tt.key)
			rr := httptest.NewRecorder()

			handler := SetConfig(testPool)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				val, err := storage.GetConfig(context.Background(), testPool, tt.namespace, tt.key)
				if err != nil {
					t.Errorf("config not saved in DB: %v", err)
				}
				if val != "test-value" {
					t.Errorf("expected value 'test-value', got %q", val)
				}
			}

			if tt.wantError != "" {
				var resp map[string]string
				_ = json.NewDecoder(rr.Body).Decode(&resp)
				if !strings.Contains(resp["error"], tt.wantError) {
					t.Errorf("expected error containing %q, got %q", tt.wantError, resp["error"])
				}
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	ctx := context.Background()

	_, _ = storage.DeleteConfig(ctx, testPool, "test-ns", "test-key")
	if err := storage.SetConfig(ctx, testPool, "test-ns", "test-key", "test-value"); err != nil {
		t.Fatalf("failed to prepare test config: %v", err)
	}
	t.Cleanup(func() {
		_, _ = storage.DeleteConfig(ctx, testPool, "test-ns", "test-key")
	})

	tests := []struct {
		name       string
		namespace  string
		key        string
		wantStatus int
		wantValue  string
		wantError  string
	}{
		{
			name:       "existing config",
			namespace:  "test-ns",
			key:        "test-key",
			wantStatus: http.StatusOK,
			wantValue:  "test-value",
		},
		{
			name:       "not found",
			namespace:  "test-ns",
			key:        "non-existent",
			wantStatus: http.StatusNotFound,
			wantError:  "config not found",
		},
		{
			name:       "invalid namespace",
			namespace:  "invalid namespace!",
			key:        "test-key",
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
		{
			name:       "invalid key",
			namespace:  "test-ns",
			key:        "invalid key!",
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/config/x/x", nil)
			req.SetPathValue("namespace", tt.namespace)
			req.SetPathValue("key", tt.key)
			rr := httptest.NewRecorder()

			handler := GetConfig(testPool)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]string
				_ = json.NewDecoder(rr.Body).Decode(&resp)
				if resp["value"] != tt.wantValue {
					t.Errorf("expected value %q, got %q", tt.wantValue, resp["value"])
				}
			} else if tt.wantError != "" {
				var resp map[string]string
				_ = json.NewDecoder(rr.Body).Decode(&resp)
				if !strings.Contains(resp["error"], tt.wantError) {
					t.Errorf("expected error containing %q, got %q", tt.wantError, resp["error"])
				}
			}
		})
	}
}

func TestListConfigs(t *testing.T) {
	ctx := context.Background()

	_, _ = storage.DeleteConfig(ctx, testPool, "test-ns", "key1")
	_, _ = storage.DeleteConfig(ctx, testPool, "test-ns", "key2")
	if err := storage.SetConfig(ctx, testPool, "test-ns", "key1", "value1"); err != nil {
		t.Fatalf("failed to prepare config: %v", err)
	}
	if err := storage.SetConfig(ctx, testPool, "test-ns", "key2", "value2"); err != nil {
		t.Fatalf("failed to prepare config: %v", err)
	}
	t.Cleanup(func() {
		_, _ = storage.DeleteConfig(ctx, testPool, "test-ns", "key1")
		_, _ = storage.DeleteConfig(ctx, testPool, "test-ns", "key2")
	})

	tests := []struct {
		name        string
		namespace   string
		wantStatus  int
		wantConfigs map[string]string
		wantError   string
	}{
		{
			name:       "existing namespace",
			namespace:  "test-ns",
			wantStatus: http.StatusOK,
			wantConfigs: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:        "empty namespace",
			namespace:   "empty-ns",
			wantStatus:  http.StatusOK,
			wantConfigs: map[string]string{},
		},
		{
			name:       "invalid namespace",
			namespace:  "invalid namespace!",
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/config/x", nil)
			req.SetPathValue("namespace", tt.namespace)
			rr := httptest.NewRecorder()

			handler := ListConfigs(testPool)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Namespace string            `json:"namespace"`
					Configs   map[string]string `json:"configs"`
				}
				_ = json.NewDecoder(rr.Body).Decode(&resp)
				if resp.Namespace != tt.namespace {
					t.Errorf("expected namespace %q, got %q", tt.namespace, resp.Namespace)
				}
				if len(resp.Configs) != len(tt.wantConfigs) {
					t.Errorf("expected %d configs, got %d", len(tt.wantConfigs), len(resp.Configs))
				}
				for k, v := range tt.wantConfigs {
					if resp.Configs[k] != v {
						t.Errorf("expected config %q=%q, got %q", k, v, resp.Configs[k])
					}
				}
			} else if tt.wantError != "" {
				var resp map[string]string
				_ = json.NewDecoder(rr.Body).Decode(&resp)
				if !strings.Contains(resp["error"], tt.wantError) {
					t.Errorf("expected error containing %q, got %q", tt.wantError, resp["error"])
				}
			}
		})
	}
}

func TestDeleteConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		namespace  string
		key        string
		wantStatus int
		wantError  string
	}{
		{
			name:       "existing config",
			namespace:  "test-ns",
			key:        "test-key",
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			namespace:  "test-ns",
			key:        "non-existent",
			wantStatus: http.StatusNotFound,
			wantError:  "Config not found",
		},
		{
			name:       "invalid namespace",
			namespace:  "invalid namespace!",
			key:        "test-key",
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
		{
			name:       "invalid key",
			namespace:  "test-ns",
			key:        "invalid key!",
			wantStatus: http.StatusBadRequest,
			wantError:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = storage.DeleteConfig(ctx, testPool, tt.namespace, tt.key)
			if tt.name == "existing config" {
				if err := storage.SetConfig(ctx, testPool, tt.namespace, tt.key, "test-value"); err != nil {
					t.Fatalf("failed to prepare config: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/x/x", nil)
			req.SetPathValue("namespace", tt.namespace)
			req.SetPathValue("key", tt.key)
			rr := httptest.NewRecorder()

			handler := DeleteConfig(testPool)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				_, err := storage.GetConfig(ctx, testPool, tt.namespace, tt.key)
				if err == nil {
					t.Errorf("expected record to be deleted, but it still exists")
				}
			}

			if tt.wantError != "" {
				var resp map[string]string
				_ = json.NewDecoder(rr.Body).Decode(&resp)
				if !strings.Contains(resp["error"], tt.wantError) {
					t.Errorf("expected error containing %q, got %q", tt.wantError, resp["error"])
				}
			}
		})
	}
}
