package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/vyacheslav-nuykin/config-service/internal/api"
	"github.com/vyacheslav-nuykin/config-service/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := storage.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := storage.RunMigrations(dbURL); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", api.Root)
	mux.HandleFunc("GET /health", api.Health(pool))
	mux.HandleFunc("GET /info", api.Info)
	mux.HandleFunc("POST /config/{namespace}/{key}", api.SetConfig(pool))
	mux.HandleFunc("GET /config/{namespace}/{key}", api.GetConfig(pool))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("[GO] Started on port: %s", port)
	log.Fatal(server.ListenAndServe())
}
