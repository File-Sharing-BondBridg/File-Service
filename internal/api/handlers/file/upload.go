package handlers

import (
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UploadResult is the per-file result object returned to the client.
type UploadResult struct {
	Success bool        `json:"success"`
	File    interface{} `json:"file,omitempty"`  // contains models.FileMetadata on success
	Error   string      `json:"error,omitempty"` // error message on failure
}

// UploadFile supports both single and multiple file uploads.
func UploadFile(c *gin.Context) {
	// Authenticate user
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		// fallback: maybe a single file
		if f, ferr := c.FormFile("file"); ferr == nil && f != nil {
			form = &multipart.Form{
				File: map[string][]*multipart.FileHeader{
					"file": {f},
				},
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form: " + err.Error()})
			return
		}
	}

	var files []*multipart.FileHeader

	// Preferred: "files"
	if fs, found := form.File["files"]; found && len(fs) > 0 {
		files = fs
	}

	// Fallback: "file"
	if len(files) == 0 {
		if f, found := form.File["file"]; found && len(f) > 0 {
			files = f
		}
	}

	// No files found
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	// Validate per-file size
	for _, fh := range files {
		if fh.Size > (200 << 20) { // 200 MB
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "file too large: " + fh.Filename,
			})
			return
		}
	}

	// Process each file
	results := make([]UploadResult, 0, len(files))

	for _, fh := range files {
		meta, err := processSingleFile(fh, userID)
		if err != nil {
			results = append(results, UploadResult{
				Success: false,
				Error:   err.Error(),
			})
		} else {
			results = append(results, UploadResult{
				Success: true,
				File:    meta,
			})
		}
	}

	// Response
	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}
