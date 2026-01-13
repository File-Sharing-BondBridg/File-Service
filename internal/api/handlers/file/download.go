package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services/query"
	"github.com/gin-gonic/gin"
)

func DownloadFile(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := query.GetFileMetadata(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	userID, _ := c.Get("user_id")
	if metadata.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
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
