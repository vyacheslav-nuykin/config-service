package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[WARNING]: .env file not found. System variables are being used.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: nil,
	}

	log.Println("[GO] Started on port: " + port)
	log.Fatal(server.ListenAndServe())
}
