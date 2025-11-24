package handlers

import (
	"fmt"
	"net/http"

	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/gin-gonic/gin"
)

func DeleteFile(c *gin.Context) {
	fileID := c.Param("id")

	// Validate file ID
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	// Get file metadata first
	metadata, exists := services.GetFileMetadata(fileID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	userID, _ := userIDFromContext(c)
	if metadata.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
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
	if services.DeleteFileMetadata(fileID, userID) {
		c.JSON(http.StatusOK, gin.H{
			"message": "File deleted successfully",
			"file_id": fileID,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file metadata"})
	}
}
