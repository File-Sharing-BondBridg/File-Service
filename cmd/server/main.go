package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/File-Sharing-BondBridg/File-Service/cmd/middleware"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api/handlers/user"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api/handlers/util"
	"github.com/File-Sharing-BondBridg/File-Service/internal/configuration"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	tracer.Start(
		tracer.WithEnv("staging"),
		tracer.WithService("file-service"),
		tracer.WithServiceVersion("1.0.0"),
	)
	defer tracer.Stop()

	// Load configuration
	cfg := configuration.Load()

	err := middleware.InitAuth(cfg.KeycloakUrl)
	if err != nil {
		log.Fatal("INIT AUTH FAILED:", err)
	}

	// Initialize PostgreSQL
	if err := services.InitializePostgres(cfg.Database.ConnectionString()); err != nil {
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
	//if err := os.MkdirAll("./temp/uploads", 0755); err != nil {
	//	log.Printf("Warning: Failed to create uploads temp directory: %v", err)
	//}
	//if err := os.MkdirAll("./temp/downloads", 0755); err != nil {
	//	log.Printf("Warning: Failed to create downloads temp directory: %v", err)
	//}
	//if err := os.MkdirAll("./temp/previews", 0755); err != nil {
	//	log.Printf("Warning: Failed to create previews temp directory: %v", err)
	//}

	setupNATS(cfg.NATSURL, cfg.CLAMAVURL)

	setupGracefulShutdown()

	r := gin.Default()

	r.Use(gintrace.Middleware("file-service"))
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
		stats := services.GetStats()
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

func setupNATS(natsUrl, clamAvUrl string) {
	_, jsCtx, err := services.ConnectNATS(natsUrl)
	if err != nil {
		log.Fatal("Failed to connect to NATS/JetStream:", err)
	}
	_ = jsCtx

	// Create/verify JetStream stream (idempotent; covers files.* and users.* subjects)
	streamCfg := &nats.StreamConfig{
		Name:      "file-events",
		Subjects:  []string{"files.*", "users.*"},
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		Discard:   nats.DiscardOld,
		MaxAge:    24 * time.Hour,
	}
	if _, err := jsCtx.AddStream(streamCfg); err != nil {
		if !strings.Contains(err.Error(), "stream name already in use") {
			log.Printf("Warning: Failed to create/verify stream: %v", err)
		} else {
			log.Println("Stream 'file-events' already exists")
		}
	} else {
		log.Println("JetStream stream 'file-events' created")
	}

	// Subscribe to files.uploaded (durable consumer)
	_, err = services.SubscribeEvent("files.uploaded", "file_service_preview", func(msg *nats.Msg) {
		log.Printf("[JetStream] files.uploaded message: %s", string(msg.Data))

		var event map[string]interface{}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Println("Invalid event:", err)
			msg.Nak()
			return
		}

		// Extract with type assertions (add more fields as needed)
		fileID, ok := event["file_id"].(string)
		if !ok {
			log.Println("Missing or invalid file_id")
			msg.Nak()
			return
		}

		filePath, ok := event["object_name"].(string) // Fixed: snake_case
		if !ok {
			log.Println("Missing or invalid object_name")
			msg.Nak()
			return
		}

		util.ScanFile(fileID, filePath, clamAvUrl)

		if err := msg.Ack(); err != nil {
			log.Printf("[JetStream] ack failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("Failed to subscribe to files.uploaded: %v", err)
	} else {
		log.Println("Subscribed to files.uploaded (durable preview consumer)")
	}

	// Subscribe to users.deleted (durable consumer)
	_, err = services.SubscribeEvent("users.deleted", "file_service_user_cleanup", func(msg *nats.Msg) {
		log.Printf("[JetStream] users.deleted message: %s", string(msg.Data))
		// Add your cleanup logic here (e.g., delete files/DB records)
		// ...

		if err := msg.Ack(); err != nil {
			log.Printf("[JetStream] ack failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("Failed to subscribe to users.deleted: %v", err)
	} else {
		log.Println("Subscribed to users.deleted (durable cleanup consumer)")
	}

	_, err = services.SubscribeEvent(
		"users.synced",
		"file_service_user_init",
		func(msg *nats.Msg) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := user.HandleUserSynced(ctx, msg.Data); err != nil {
				log.Printf("[JetStream] user sync failed: %v", err)
				_ = msg.Nak()
				return
			}

			if err := msg.Ack(); err != nil {
				log.Printf("[JetStream] ack failed: %v", err)
			}
		},
	)
	if err != nil {
		log.Printf("Failed to subscribe to users.synced: %v", err)
	} else {
		log.Println("Subscribed to users.synced (durable cleanup consumer)")
	}

	log.Println("NATS/JetStream setup completed")
}
