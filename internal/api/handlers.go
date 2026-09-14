package api

import (
	"encoding/json"
	"net/http"
	"os"
)

// For now, we return an error for anything that doesn't relate to our handler.
func Root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
		version = "0.1.0"
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"service": serviceName,
		"version": version,
	})
}
