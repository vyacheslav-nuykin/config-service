package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vyacheslav-nuykin/config-service/internal/storage"
)

var validNamespace = regexp.MustCompile(`^[a-z0-9_-]+$`)
var validKey = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

type ValidationError struct {
	ErrorStr string `json:"error"`
	Field    string `json:"field"`
	Reason   string `json:"reason"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: field %s %s", e.ErrorStr, e.Field, e.Reason)
}

func validateNamespace(namespace string) error {
	if namespace == "" {
		return &ValidationError{ErrorStr: "validation failed", Field: "namespace", Reason: "cannot be empty"}
	}
	if !validNamespace.MatchString(namespace) {
		return &ValidationError{ErrorStr: "validation failed", Field: "namespace", Reason: "contains invalid characters"}
	}
	if len(namespace) > 64 {
		return &ValidationError{ErrorStr: "validation failed", Field: "namespace", Reason: "length must be between 1 and 64"}
	}
	return nil
}

func validateKey(key string) error {
	if key == "" {
		return &ValidationError{ErrorStr: "validation failed", Field: "key", Reason: "cannot be empty"}
	}
	if !validKey.MatchString(key) {
		return &ValidationError{ErrorStr: "validation failed", Field: "key", Reason: "contains invalid characters"}
	}
	if len(key) > 128 {
		return &ValidationError{ErrorStr: "validation failed", Field: "key", Reason: "length must be between 1 and 128"}
	}
	return nil
}

func validateValue(value string) error {
	if len(value) > 4096 {
		return &ValidationError{ErrorStr: "validation failed", Field: "value", Reason: "length must be between 0 and 4096"}
	}
	return nil
}

// writeValidationError writes a ValidationError to the response as JSON.
func writeValidationError(w http.ResponseWriter, err error) {
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(valErr)
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// For now, we return an error for anything that doesn't relate to our handler.
func Root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
}

func Health(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status":   "degraded",
				"database": "unreachable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "ok",
			"database": "ok",
		})
	}
}

func Info(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	serviceName := os.Getenv("SERVICE")
	if serviceName == "" {
		serviceName = "config-service"
	}

	version := os.Getenv("VERSION")
	if version == "" {
		version = "0.0.0"
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"service": serviceName,
		"version": version,
	})
}

func SetConfig(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		namespace := r.PathValue("namespace")
		key := r.PathValue("key")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		var req struct {
			Value string `json:"value"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		if err := validateNamespace(namespace); err != nil {
			writeValidationError(w, err)
			return
		}
		if err := validateKey(key); err != nil {
			writeValidationError(w, err)
			return
		}
		if err := validateValue(req.Value); err != nil {
			writeValidationError(w, err)
			return
		}

		err = storage.SetConfig(ctx, pool, namespace, key, req.Value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}

func GetConfig(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		namespace := r.PathValue("namespace")
		key := r.PathValue("key")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := validateNamespace(namespace); err != nil {
			writeValidationError(w, err)
			return
		}
		if err := validateKey(key); err != nil {
			writeValidationError(w, err)
			return
		}

		value, err := storage.GetConfig(ctx, pool, namespace, key)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "config not found",
				})
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"namespace": namespace,
			"key":       key,
			"value":     value,
		})
	}
}

func ListConfigs(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		namespace := r.PathValue("namespace")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := validateNamespace(namespace); err != nil {
			writeValidationError(w, err)
			return
		}

		configs, err := storage.ListConfigs(ctx, pool, namespace)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		if configs == nil {
			configs = make(map[string]string)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"namespace": namespace,
			"configs":   configs,
		})
	}
}

func DeleteConfig(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		namespace := r.PathValue("namespace")
		key := r.PathValue("key")

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := validateNamespace(namespace); err != nil {
			writeValidationError(w, err)
			return
		}
		if err := validateKey(key); err != nil {
			writeValidationError(w, err)
			return
		}

		rowsAffected, err := storage.DeleteConfig(ctx, pool, namespace, key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if rowsAffected == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Config not found"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}
