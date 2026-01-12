package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/File-Sharing-BondBridg/File-Service/internal/models"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/gin-gonic/gin"
)

func GetFile(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := services.GetFileMetadata(id)
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

	c.File(tempPath)
}

func ListFiles(c *gin.Context) {
	userID, exists := userIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse pagination params
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "50")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 50
	}
	// Cap page size to avoid abuse
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize

	files, err := services.GetUserFileMetadataPage(userID, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files"})
		return
	}
	total, err := services.GetUserFileCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total count"})
		return
	}
	// Compute total pages
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"files":      files,
		"page":       page,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
	})
}

func GetFileMetadataForUser(fileID, userID string) (models.FileMetadata, bool) {
	pg := getPostgresForUser(userID)
	return pg.getFileMetadata(fileID)
}

func GetPreview(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := services.GetFileMetadata(id)
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
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {

		}
	}(tempPath) // Clean up after serving

	if err := minioService.DownloadFile(metadata.PreviewPath, tempPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preview file not found"})
		return
	}

	c.File(tempPath)
}

func GetFileInfo(c *gin.Context) {
	id := c.Param("id")

	// Get file metadata
	metadata, exists := services.GetFileMetadata(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	userID, _ := c.Get("user_id")
	if metadata.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file": metadata,
	})
}
