package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/File-Sharing-BondBridg/File-Service/internal/configuration"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/File-Sharing-BondBridg/File-Service/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize PostgreSQL
	if err := storage.InitializePostgres(cfg.Database.ConnectionString()); err != nil {
		log.Fatalf("❌ Failed to initialize PostgreSQL: %v", err)
	}

	// Initialize MinIO
	if err := services.InitializeMinio(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.BucketName,
		cfg.MinIO.UseSSL,
	); err != nil {
		log.Fatalf("❌ Failed to initialize MinIO: %v", err)
	}

	// Create necessary directories
	if err := os.MkdirAll("./temp", 0755); err != nil {
		log.Printf("Warning: Failed to create temp directory: %v", err)
	}
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		log.Printf("Warning: Failed to create uploads directory: %v", err)
	}

	// Set up graceful shutdown
	setupGracefulShutdown()

	// Setup router
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Register routes
	api.RegisterRoutes(r)

	// Add a health check that tests all services
	r.GET("/health", func(c *gin.Context) {
		// Test database connection
		dbStats := storage.GetPostgresStorage().GetStats()
		
		// Test MinIO connection by trying to list buckets
		minioService := services.GetMinioService()
		healthStatus := "healthy"
		
		_, err := minioService.Client.BucketExists(c, cfg.MinIO.BucketName)
		if err != nil {
			healthStatus = "degraded"
			log.Printf("MinIO health check failed: %v", err)
		}

		c.JSON(200, gin.H{
			"status":   healthStatus,
			"database": "connected",
			"storage":  "connected", 
			"stats":    dbStats,
		})
	})

	log.Printf("🚀 Server starting on :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")
		
		// Close any connections if needed
		if rabbitmq := services.GetRabbitMQService(); rabbitmq != nil {
			rabbitmq.Close()
		}
		
		os.Exit(0)
	}()
}