package api

import (
	"github.com/File-Sharing-BondBridg/File-Service/cmd/middleware"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api/handlers/file"
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

func RegisterRoutes(r *gin.RouterGroup) {
	// Enable CORS for preflight requests
	r.Use(corsMiddleware())
	r.Use(middleware.RequireAuth())

	//api := r.Group("/api")
	//api.Use(middleware.RequireAuth())
	//{
	//}
	r.GET("/files/health", handlers.HealthCheck)

	// File endpoints
	r.POST("/files/upload", handlers.UploadFile) // upload a file

	r.GET("/files", handlers.ListFiles)            // list all uploaded files
	r.GET("/files/:id", handlers.GetFile)          // Get single file
	r.GET("/files/:id/info", handlers.GetFileInfo) // Get file metadata

	// Download a specific file
	r.GET("/preview/:id", handlers.GetPreview)          // fetch preview of a file
	r.GET("/files/:id/download", handlers.DownloadFile) // Download file
	r.GET("/files/:id/preview", handlers.GetPreview)    // Get preview
	r.DELETE("/files/:id/delete", handlers.DeleteFile)  // Delete file
}
