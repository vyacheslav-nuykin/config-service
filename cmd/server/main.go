package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", api.Root)
	mux.HandleFunc("GET /health", api.Health(pool))
	mux.HandleFunc("GET /info", api.Info)
	mux.HandleFunc("POST /config/{namespace}/{key}", api.SetConfig(pool))
	mux.HandleFunc("GET /config/{namespace}/{key}", api.GetConfig(pool))
	mux.HandleFunc("GET /config/{namespace}", api.ListConfigs(pool))
	mux.HandleFunc("DELETE /config/{namespace}/{key}", api.DeleteConfig(pool))

	stopCtx, stopCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopCancel()

	handler := api.Logger(mux)
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		log.Printf("[GO] Started on port: %s", port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stopCtx.Done()
	log.Println("[GO] Shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server Shutdown error: %v", err)
	}

	defer pool.Close()
	log.Println("[GO] Server stopped. Connections closed.")
}
