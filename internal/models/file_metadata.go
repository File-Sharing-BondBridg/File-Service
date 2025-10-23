package models

import (
	"time"
)

type FileMetadata struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"original_name"`
	Size         int64     `json:"size"`
	Type         string    `json:"type"`
	Extension    string    `json:"extension"`
	UploadedAt   time.Time `json:"uploaded_at"`
	FilePath     string    `json:"file_path"`
	PreviewPath  string    `json:"preview_path"`
	ShareURL     string    `json:"share_url"`
	BucketName   string    `json:"bucket_name"`
}
