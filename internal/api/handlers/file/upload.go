package handlers

import (
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UploadResult is the per-file result object returned to the client.
type UploadResult struct {
	Success bool        `json:"success"`
	File    interface{} `json:"file,omitempty"`  // will contain models.FileMetadata on success
	Error   string      `json:"error,omitempty"` // error message on failure
}

// UploadFile supports both single and multiple file uploads.
// - Accepts form field "files" (multiple) or "file" (single) for backward compatibility.
// - Returns detailed per-file results (Option B).
func UploadFile(c *gin.Context) {
	// Authenticate user from context
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	// Parse multipart form files. Support both "files" (array) and "file" (single fallback).
	form, err := c.MultipartForm()
	if err != nil {
		// Fallback: maybe a single file in "file" field or c.FormFile
		if f, ferr := c.FormFile("file"); ferr == nil && f != nil {
			// single-file fallback handled below
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form: " + err.Error()})
			return
		}
	}

	var files []*multipart.FileHeader

	// prefer "files" (multiple)
	if form != nil {
		if fs, found := form.File["files"]; found && len(fs) > 0 {
			files = fs
		}
	}

	// fallback to single named "file"
	if len(files) == 0 {
		if f, ferr := c.FormFile("file"); ferr == nil && f != nil {
			files = []*multipart.FileHeader{f}
		}
	}

	// No files provided
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	// Validate global per-request constraints (optional: enforce user quotas, total size, etc.)
	// Example: check each file size (we support up to 200MB per file in your requirements)
	for _, fh := range files {
		if fh.Size > (200 << 20) { // 200 MB
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "file too large: " + fh.Filename,
			})
			return
		}
	}

	// Process each file using the shared pipeline
	results := make([]UploadResult, 0, len(files))
	for _, fh := range files {
		meta, err := processSingleFile(c, fh, userID)
		if err != nil {
			results = append(results, UploadResult{
				Success: false,
				Error:   err.Error(),
			})
			continue
		}
		results = append(results, UploadResult{
			Success: true,
			File:    meta,
		})
	}

	// Return an array of per-file results (Option B)
	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}
