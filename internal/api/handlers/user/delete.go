package user

import (
	"encoding/json"
	"log"

	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services/command"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services/query"
	"github.com/nats-io/nats.go"
)

type UserDeletedPayload struct {
	UserID string `json:"user_id"`
}

func HandleUserDeleted(msg *nats.Msg) {
	var payload UserDeletedPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("[NATS] users.deleted: invalid JSON: %v", err)
		nak(msg)
		return
	}

	userID := payload.UserID
	if userID == "" {
		log.Printf("[NATS] users.deleted: missing user_id")
		nak(msg)
		return
	}

	log.Printf("[NATS] Processing users.deleted for user_id: %s", userID)

	// 1. Get all file paths from DB
	filePaths, err := query.GetFilePathsForUser(userID)
	if err != nil {
		log.Printf("[NATS] Failed to get file paths: %v", err)
		nak(msg)
		return
	}

	if len(filePaths) == 0 {
		log.Printf("[NATS] No files found for user %s", userID)
	} else {
		log.Printf("[NATS] Found %d files to delete: %v", len(filePaths), filePaths)
	}

	// 2. Delete from MinIO
	minioSvc := services.GetMinioService()
	if minioSvc == nil {
		log.Printf("[NATS] MinIO service not available")
	} else {
		for _, path := range filePaths {
			if err := minioSvc.DeleteFile(path); err != nil {
				log.Printf("[NATS] Failed to delete MinIO object %s: %v", path, err)
				nak(msg)
				return
			}
		}
		log.Printf("[NATS] Deleted %d MinIO objects", len(filePaths))
	}

	// 3. Delete from PostgreSQL
	deletedCount := command.DeleteAllFilesForUser(userID)
	log.Printf("[NATS] Deleted %d file records from DB", deletedCount)

	log.Printf("[NATS] Successfully cleaned up user %s", userID)
	ack(msg)
}

// ack safely acknowledges the message
func ack(msg *nats.Msg) {
	if err := msg.Ack(); err != nil {
		log.Printf("[NATS] Failed to ack message: %v", err)
	}
}

// nak negatively acknowledges (retry)
func nak(msg *nats.Msg) {
	if err := msg.Nak(); err != nil {
		log.Printf("[NATS] Failed to nak message: %v", err)
	}
}
