package handlers

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

type FileUploadedEvent struct {
	FileID     string `json:"file_id"`
	ObjectName string `json:"objectName"`
	FileType   string `json:"fileType"`
}

type UserDeletedEvent struct {
	UserID string `json:"user_id"`
}

type FileDeletedEvent struct {
	FileID string `json:"file_id"`
}

func HandleFileUploaded(msg *nats.Msg) {
	log.Println("[NATS] Received files.uploaded")

	var payload FileUploadedEvent
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("[NATS] files.uploaded: invalid payload: %v", err)
		_ = msg.Nak()
		return
	}

	log.Printf("[NATS] File uploaded: %s (%s)", payload.FileID, payload.FileType)

	// TODO: preview worker, virus scanner, audit logs
	// Example:
	// if payload.FileType == "image" {
	//     preview.Generate(payload.ObjectName)
	// }

	_ = msg.Ack()
}

//func HandleFileDeleted(msg *nats.Msg) {
//	log.Println("[NATS] Received files.deleted")
//
//	var payload FileDeletedEvent
//	if err := json.Unmarshal(msg.Data, &payload); err != nil {
//		log.Printf("[NATS] files.deleted: invalid payload: %v", err)
//		_ = msg.Nak()
//		return
//	}
//
//	log.Printf("[NATS] Removing file (DB + Storage): %s", payload.FileID)
//
//	err := services.DeleteFileCompletely(payload.FileID)
//	if err != nil {
//		log.Printf("[NATS] files.deleted cleanup failed: %v", err)
//		_ = msg.Nak()
//		return
//	}
//
//	_ = msg.Ack()
//	log.Printf("[NATS] File cleanup completed: %s", payload.FileID)
//}
