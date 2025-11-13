package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/File-Sharing-BondBridg/File-Service/cmd/middleware"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api/handlers"
	"github.com/File-Sharing-BondBridg/File-Service/internal/configuration"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/File-Sharing-BondBridg/File-Service/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

func main() {
	// Load configuration
	cfg := configuration.Load()

	err := middleware.InitAuth("http://localhost:8081/realms/bondbridg")
	if err != nil {
		log.Fatal("INIT AUTH FAILED:", err)
	}

	// Initialize PostgreSQL
	if err := storage.InitializePostgres(cfg.Database.ConnectionString()); err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}

	log.Printf("PostgreSQL initialized successfully")

	// Initialize MinIO
	if err := services.InitializeMinio(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.BucketName,
		cfg.MinIO.UseSSL,
	); err != nil {
		log.Fatalf("Failed to initialize MinIO: %v", err)
	}
	log.Printf("MinIO initialized successfully")

	// Create necessary temp directories
	if err := os.MkdirAll("./temp/uploads", 0755); err != nil {
		log.Printf("Warning: Failed to create uploads temp directory: %v", err)
	}
	if err := os.MkdirAll("./temp/downloads", 0755); err != nil {
		log.Printf("Warning: Failed to create downloads temp directory: %v", err)
	}
	if err := os.MkdirAll("./temp/previews", 0755); err != nil {
		log.Printf("Warning: Failed to create previews temp directory: %v", err)
	}

	setupNATS()

	setupGracefulShutdown()

	r := gin.Default()

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

	r.GET("/health", func(c *gin.Context) {
		stats := storage.GetStats()
		minioService := services.GetMinioService()
		var minioStatus string

		if minioService != nil {
			// Test MinIO connection
			if err := minioService.CheckConnection(); err == nil {
				minioStatus = "connected"
			} else {
				minioStatus = "degraded"
			}
		} else {
			minioStatus = "failed"
		}

		c.JSON(200, gin.H{
			"status": "ok",
			"storage": gin.H{
				"postgres": "connected",
				"minio":    minioStatus,
			},
			"stats": stats,
		})
	})

	apiGroup := r.Group("/api")
	api.RegisterRoutes(apiGroup)

	log.Printf("Server starting on :%s", cfg.Server.Port)
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
		os.Exit(0)
	}()
}

func setupNATS() {
	// Connect once at startup and keep it alive
	_, err := services.ConnectNATS(nats.DefaultURL)
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	log.Println("Connected to NATS")

	_, err = services.SubscribeNATS("test.subject", func(msg *nats.Msg) {
		log.Printf("Test message received: %s", string(msg.Data))
	})
	if err != nil {
		log.Println("Failed to subscribe to test.subject:", err)
	} else {
		log.Println("Subscribed to test.subject")
	}

	subs := map[string]nats.MsgHandler{
		"uploads.minio": func(msg *nats.Msg) {
			log.Printf("[MinIO] Upload event received: %s", msg.Data)
		},
		"uploads.postgres": func(msg *nats.Msg) {
			log.Printf("[Postgres] Metadata event received: %s", msg.Data)
		},
		"uploads.sync": func(msg *nats.Msg) {
			log.Printf("[Sync] Checking consistency for: %s", msg.Data)
		},
		"user.deleted": handlers.HandleUserDeleted,
	}

	for subject, handler := range subs {
		_, err := services.SubscribeNATS(subject, handler)
		if err != nil {
			log.Printf("Failed to subscribe to %s: %v", subject, err)
		} else {
			log.Printf("Subscribed to %s", subject)
		}
	}
}
