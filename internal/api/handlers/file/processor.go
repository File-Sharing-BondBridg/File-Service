package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/File-Sharing-BondBridg/File-Service/internal/models"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services/command"
	"github.com/google/uuid"
)

func processSingleFile(fileHeader *multipart.FileHeader, userID string) (models.FileMetadata, error) {

	// Generate file identifiers
	fileID := uuid.New().String()
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	fileName := strings.TrimSuffix(fileHeader.Filename, ext)

	// Determine a file type
	fileType := "other"
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		fileType = "image"
	case ".pdf", ".doc", ".docx", ".txt":
		fileType = "document"
	case ".mp4", ".avi", ".mov", ".mkv":
		fileType = "video"
	case ".mp3", ".wav", ".ogg":
		fileType = "audio"
	}

	file, err := fileHeader.Open()
	if err != nil {
		return models.FileMetadata{}, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	minioService := services.GetMinioService()
	if minioService == nil {
		return models.FileMetadata{}, fmt.Errorf("storage service not available")
	}

	objectName := fileID + ext
	contentType := services.GetContentType(ext)

	// Upload to MinIO
	if err := minioService.UploadFile(file, fileHeader.Size, objectName, contentType); err != nil {
		return models.FileMetadata{}, fmt.Errorf("failed to upload to storage: %w", err)
	}

	// Build metadata
	fileMetadata := models.FileMetadata{
		ID:           fileID,
		Name:         fileName,
		OriginalName: fileHeader.Filename,
		Size:         fileHeader.Size,
		Type:         fileType,
		Extension:    ext,
		UploadedAt:   time.Now(),
		FilePath:     objectName,
		ShareURL:     "",
		UserID:       userID,
	}

	// Save metadata
	if err := command.SaveFileMetadata(fileMetadata); err != nil {
		// cleanup MinIO object
		if delErr := minioService.DeleteFile(objectName); delErr != nil {
			log.Printf("warning: failed to cleanup object after metadata save failure: %v", delErr)
		}
		return models.FileMetadata{}, fmt.Errorf("failed to save file metadata: %w", err)
	}

	// Publish event: "files.uploaded"
	uploadEvent := map[string]interface{}{
		"action":      "uploaded",
		"file_id":     fileMetadata.ID,
		"object_name": objectName,
		"file_type":   fileType,
		"size":        fileMetadata.Size,
		"user_id":     fileMetadata.UserID,
		"uploaded_at": fileMetadata.UploadedAt.UTC().Format(time.RFC3339),
	}

	if err := services.PublishEvent("files.uploaded", uploadEvent); err != nil {
		log.Printf("warning: failed to publish files.uploaded event: %v", err)
	}

	// Publish virus scan event
	scanEvent := map[string]interface{}{
		"file_id":      fileID,
		"object_name":  objectName,
		"user_id":      userID,
		"requested_at": time.Now().UTC().Format(time.RFC3339),
	}

	if err := services.PublishPlain("files.scan.requested", mustJSON(scanEvent)); err != nil {
		log.Printf("warning: failed to publish files.scan.requested command: %v", err)
	}

	return fileMetadata, nil
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
