package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/vyacheslav-nuykin/config-service/internal/api"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[WARNING]: .env file not found. System variables are being used.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", api.RootHandler)
	mux.HandleFunc("GET /health", api.HealthHandler)


	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("[GO] Started on port: %s", port)
	log.Fatal(server.ListenAndServe())
}
