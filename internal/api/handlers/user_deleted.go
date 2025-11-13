// internal/nats/handlers/user_deleted.go
package handlers

import (
	"encoding/json"
	"log"

	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/File-Sharing-BondBridg/File-Service/internal/storage"
	"github.com/nats-io/nats.go"
)

type UserDeletedPayload struct {
	UserID string `json:"user_id"`
}

func HandleUserDeleted(msg *nats.Msg) {
	var payload UserDeletedPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("[NATS] user.deleted: invalid JSON: %v", err)
		nak(msg)
		return
	}

	if payload.UserID == "" {
		log.Printf("[NATS] user.deleted: missing user_id")
		nak(msg)
		return
	}

	log.Printf("[NATS] Processing user.deleted for user_id: %s", payload.UserID)

	// 1. Delete from PostgreSQL
	deletedCount := storage.DeleteAllFilesForUser(payload.UserID)
	log.Printf("[NATS] Deleted %d file records for user %s", deletedCount, payload.UserID)

	// 2. Delete from MinIO
	minioSvc := services.GetMinioService()
	if minioSvc == nil {
		log.Printf("[NATS] MinIO service not available — skipping object deletion")
	} else {
		prefix := payload.UserID + "/" // e.g. "123e4567-e89b-12d3-a456-426614174000/"
		if err := minioSvc.DeleteObjectsByPrefix(prefix); err != nil {
			log.Printf("[NATS] Failed to delete MinIO objects for user %s: %v", payload.UserID, err)
			nak(msg)
			return
		}
		log.Printf("[NATS] Deleted MinIO objects with prefix: %s", prefix)
	}

	log.Printf("[NATS] Successfully cleaned up user %s", payload.UserID)
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
