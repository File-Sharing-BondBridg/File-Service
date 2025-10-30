package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/File-Sharing-BondBridg/File-Service/internal/models"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/File-Sharing-BondBridg/File-Service/internal/storage"
	uploads "github.com/File-Sharing-BondBridg/File-Service/uploads/previews"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Generate unique file ID
	fileID := uuid.New().String()

	// Get file extension and type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	fileName := strings.TrimSuffix(file.Filename, ext)

	// Determine file type
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

	// Save file locally first (temporary)
	tempLocalPath := fmt.Sprintf("./temp/uploads/%s%s", fileID, ext)
	if err := os.MkdirAll("./temp/uploads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	if err := c.SaveUploadedFile(file, tempLocalPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Always clean up the temporary local file
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Println("Error removing temp file")
		}
	}(tempLocalPath)

	// Get MinIO service
	minioService := services.GetMinioService()
	if minioService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service not available"})
		return
	}

	// Upload to MinIO
	objectName := fileID + ext
	contentType := services.GetContentType(ext)

	if err := minioService.UploadFile(tempLocalPath, objectName, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to storage: " + err.Error()})
		return
	}

	// Generate preview for supported file types and upload to MinIO
	var previewObjectName string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		previewPath, err := uploads.GenerateImagePreview(tempLocalPath, 200)
		if err == nil && previewPath != "" {
			previewObjectName = "previews/" + fileID + ".jpg"
			if err := minioService.UploadFile(previewPath, previewObjectName, "image/jpeg"); err != nil {
				fmt.Printf("Warning: Failed to upload preview: %v\n", err)
			}
			// Clean up local preview
			defer os.Remove(previewPath)
		}
	case ".pdf":
		previewPath, err := uploads.GeneratePDFPreview(tempLocalPath, 200)
		if err == nil && previewPath != "" {
			previewObjectName = "previews/" + fileID + ".jpg"
			if err := minioService.UploadFile(previewPath, previewObjectName, "image/jpeg"); err != nil {
				fmt.Printf("Warning: Failed to upload preview: %v\n", err)
			}
			// Clean up local preview
			defer os.Remove(previewPath)
		}
	}

	// Create file metadata
	fileMetadata := models.FileMetadata{
		ID:           fileID,
		Name:         fileName,
		OriginalName: file.Filename,
		Size:         file.Size,
		Type:         fileType,
		Extension:    ext,
		UploadedAt:   time.Now(),
		FilePath:     objectName,        // MinIO object name
		PreviewPath:  previewObjectName, // MinIO preview object name
		ShareURL:     "",                // Will be set when shared
	}

	// Save metadata to PostgreSQL
	if err := storage.SaveFileMetadata(fileMetadata); err != nil {
		// Clean up files from MinIO if metadata save fails
		minioService.DeleteFile(objectName)
		if previewObjectName != "" {
			minioService.DeleteFile(previewObjectName)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
		return
	}

	// Publish message to RabbitMQ for async processing (if needed)
	rabbitmqService := services.GetRabbitMQService()
	if rabbitmqService != nil {
		message := services.FileProcessingMessage{
			FileID:    fileID,
			FilePath:  objectName,
			FileType:  fileType,
			Operation: "upload",
		}
		rabbitmqService.PublishFileProcessingMessage(message)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"file":    fileMetadata,
		"storage": "minio",
	})
}

func GetFile(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := storage.GetFileMetadata(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Get MinIO service
	minioService := services.GetMinioService()
	if minioService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service not available"})
		return
	}

	// Download from MinIO to temporary location
	tempPath := fmt.Sprintf("./temp/downloads/%s%s", metadata.ID, metadata.Extension)
	if err := os.MkdirAll("./temp/downloads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}
	defer os.Remove(tempPath) // Clean up after serving

	if err := minioService.DownloadFile(metadata.FilePath, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download file from storage"})
		return
	}

	c.File(tempPath)
}

func ListFiles(c *gin.Context) {
	// Get all files from PostgreSQL
	files := storage.GetAllFileMetadata()

	c.JSON(http.StatusOK, gin.H{
		"files": files,
	})
}

func GetPreview(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := storage.GetFileMetadata(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Check if preview exists
	if metadata.PreviewPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preview not available for this file type"})
		return
	}

	// Get MinIO service
	minioService := services.GetMinioService()
	if minioService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service not available"})
		return
	}

	// Download preview from MinIO to temporary location
	tempPath := fmt.Sprintf("./temp/previews/%s.jpg", id)
	if err := os.MkdirAll("./temp/previews", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}
	defer os.Remove(tempPath) // Clean up after serving

	if err := minioService.DownloadFile(metadata.PreviewPath, tempPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preview file not found"})
		return
	}

	c.File(tempPath)
}

func DeleteFile(c *gin.Context) {
	fileID := c.Param("id")

	// Validate file ID
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	// Get file metadata first
	metadata, exists := storage.GetFileMetadata(fileID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Get MinIO service
	minioService := services.GetMinioService()
	if minioService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service not available"})
		return
	}

	// Delete from MinIO
	if err := minioService.DeleteFile(metadata.FilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file from storage: " + err.Error()})
		return
	}

	// Delete preview from MinIO if it exists
	if metadata.PreviewPath != "" {
		if err := minioService.DeleteFile(metadata.PreviewPath); err != nil {
			fmt.Printf("Warning: Failed to delete preview from storage: %v\n", err)
		}
	}

	// Delete metadata from PostgreSQL
	if storage.DeleteFileMetadata(fileID) {
		c.JSON(http.StatusOK, gin.H{
			"message": "File deleted successfully",
			"file_id": fileID,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file metadata"})
	}
}

func DownloadFile(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := storage.GetFileMetadata(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Get MinIO service
	minioService := services.GetMinioService()
	if minioService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service not available"})
		return
	}

	// Download from MinIO to temporary location
	tempPath := fmt.Sprintf("./temp/downloads/%s%s", metadata.ID, metadata.Extension)
	if err := os.MkdirAll("./temp/downloads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}
	defer os.Remove(tempPath) // Clean up after serving

	if err := minioService.DownloadFile(metadata.FilePath, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download file from storage"})
		return
	}

	// Set appropriate headers for download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+metadata.OriginalName)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Transfer-Encoding", "binary")

	c.File(tempPath)
}

func GetFileInfo(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := storage.GetFileMetadata(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file": metadata,
	})
}
