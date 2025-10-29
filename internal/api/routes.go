package api

import (
	"github.com/File-Sharing-BondBridg/File-Service/internal/api/handlers"
	"github.com/gin-gonic/gin"
)

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PATCH, PUT, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	}
}

func RegisterRoutes(r *gin.Engine) {
	// Enable CORS for preflight requests
	r.Use(corsMiddleware())

	api := r.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)

		// File endpoints
		api.POST("/upload", handlers.UploadFile)         // upload a file
		api.GET("/files", handlers.ListFiles)            // list all uploaded files
		api.GET("/files/:id", handlers.GetFile)          // Get single file
		api.GET("/files/:id/info", handlers.GetFileInfo) // Get file metadata

		// download a specific file
		api.GET("/preview/:id", handlers.GetPreview)          // fetch preview of a file
		api.GET("/files/:id/download", handlers.DownloadFile) // Download file
		api.GET("/files/:id/preview", handlers.GetPreview)    // Get preview
		api.DELETE("/files/:id/delete", handlers.DeleteFile)  // Delete file
	}
}
