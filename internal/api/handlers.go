package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vyacheslav-nuykin/config-service/internal/storage"
)

// For now, we return an error for anything that doesn't relate to our handler.
func Root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
}

func Health(pool *pgxpool.Pool, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	serviceStatus := "ok"
	dbStatus := "ok"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := storage.New(ctx, os.Getenv("DATABASE_URL"))

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		serviceStatus = "degraded"
		dbStatus = "unreachable"
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   serviceStatus,
		"database": dbStatus,
	})
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
