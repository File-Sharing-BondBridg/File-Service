package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/File-Sharing-BondBridg/File-Service/internal/models"
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

	// Create destination path with unique ID
	dst := fmt.Sprintf("./uploads/%s%s", fileID, ext)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate preview for supported file types
	var previewPath string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		previewPath, _ = uploads.GenerateImagePreview(dst, 200)
	case ".pdf":
		previewPath, _ = uploads.GeneratePDFPreview(dst, 200)
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
		FilePath:     dst,
		PreviewPath:  previewPath,
		ShareURL:     "", // Will be set when shared
	}

	// Save metadata to database
	if err := storage.SaveFileMetadata(fileMetadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"file":    fileMetadata,
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

	// Serve the actual file
	c.File(metadata.FilePath)
}

func ListFiles(c *gin.Context) {
	// Get all files from metadata store
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

	if _, err := os.Stat(metadata.PreviewPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preview file not found"})
		return
	}

	c.File(metadata.PreviewPath)
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

	// Delete the actual file
	if err := os.Remove(metadata.FilePath); err != nil {
		if !os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file: " + err.Error()})
			return
		}
		// File doesn't exist, but we'll continue to delete metadata
	}

	// Delete preview file if it exists
	if metadata.PreviewPath != "" {
		if err := os.Remove(metadata.PreviewPath); err != nil {
			if !os.IsNotExist(err) {
				// Log the error but don't fail the entire operation
				fmt.Printf("Warning: Failed to delete preview file: %v\n", err)
			}
		}
	}

	// Delete metadata from storage
	if storage.DeleteFileMetadata(fileID) {
		c.JSON(http.StatusOK, gin.H{
			"message": "File deleted successfully",
			"file_id": fileID,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file metadata"})
	}
}

// DownloadFile Add this method for downloading files
func DownloadFile(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := storage.GetFileMetadata(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Set appropriate headers for download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+metadata.OriginalName)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Transfer-Encoding", "binary")

	c.File(metadata.FilePath)
}

// GetFileInfo Add this method for getting file info
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
