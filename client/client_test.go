package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/vyacheslav-nuykin/config-service/client"
)

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/config/my-ns/my-key" {
			t.Errorf("path = %q, want /api/v1/config/my-ns/my-key", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"namespace": "my-ns",
			"key":       "my-key",
			"value":     "my-value",
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	got, err := c.Get(context.Background(), "my-ns", "my-key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "my-value" {
		t.Errorf("Get = %q, want %q", got, "my-value")
	}
}

func TestClient_Get_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "config not found"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.Get(context.Background(), "my-ns", "missing")
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestClient_Get_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "boom"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := c.Get(context.Background(), "my-ns", "my-key")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if errors.Is(err, client.ErrNotFound) {
		t.Errorf("500 should not be ErrNotFound, got %v", err)
	}
}

func TestClient_Set(t *testing.T) {
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/config/my-ns/my-key" {
			t.Errorf("path = %q, want /api/v1/config/my-ns/my-key", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	if err := c.Set(context.Background(), "my-ns", "my-key", "my-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := map[string]string{"value": "my-value"}
	if !reflect.DeepEqual(gotBody, want) {
		t.Errorf("body = %v, want %v", gotBody, want)
	}
}

func TestClient_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/config/my-ns" {
			t.Errorf("path = %q, want /api/v1/config/my-ns", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace": "my-ns",
			"configs": map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	got, err := c.List(context.Background(), "my-ns")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := map[string]string{"k1": "v1", "k2": "v2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List = %v, want %v", got, want)
	}
}

func TestClient_List_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace": "empty-ns",
			"configs":   map[string]string{},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	got, err := c.List(context.Background(), "empty-ns")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Fatal("List returned nil map, want empty non-nil")
	}
	if len(got) != 0 {
		t.Errorf("List = %v, want empty", got)
	}
}

func TestClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/config/my-ns/my-key" {
			t.Errorf("path = %q, want /api/v1/config/my-ns/my-key", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	if err := c.Delete(context.Background(), "my-ns", "my-key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestClient_Delete_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Config not found"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	err := c.Delete(context.Background(), "my-ns", "missing")
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
