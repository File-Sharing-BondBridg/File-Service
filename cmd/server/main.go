package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/File-Sharing-BondBridg/File-Service/internal/api"
	"github.com/File-Sharing-BondBridg/File-Service/internal/storage"
	"github.com/gin-gonic/gin"
	// ... other imports
)

func main() {
	// Initialize storage (load from JSON file)
	if err := storage.Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize storage: %v", err)
		log.Println("Starting with empty file metadata...")
	}

	// Set up graceful shutdown to ensure data is saved
	setupGracefulShutdown()

	// Your existing Gin setup...
	r := gin.Default()

	api.RegisterRoutes(r)

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")
		// Storage is auto-saved on every operation, so no need to save here
		os.Exit(0)
	}()
}
