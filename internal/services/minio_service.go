package services

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	client     *minio.Client
	bucketName string
}

var minioInstance *MinioService

func InitializeMinio(endpoint, accessKey, secretKey, bucket string, useSSL bool) error {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %v", err)
	}

	// Create bucket if it doesn't exist
	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %v", err)
	}

	if !exists {
		err = client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %v", err)
		}
		log.Printf("Created bucket: %s", bucket)
	}

	minioInstance = &MinioService{
		client:     client,
		bucketName: bucket,
	}

	log.Println("Connected to MinIO successfully")
	return nil
}

func GetMinioService() *MinioService {
	return minioInstance
}

// CheckConnection Add this method for health checks
func (m *MinioService) CheckConnection() error {
	if m == nil || m.client == nil {
		return fmt.Errorf("minio service not initialized")
	}
	_, err := m.client.BucketExists(context.Background(), m.bucketName)
	return err
}

func (m *MinioService) UploadFile(localFilePath, objectName, contentType string) error {
	_, err := m.client.FPutObject(context.Background(), m.bucketName, objectName, localFilePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (m *MinioService) DownloadFile(objectName, localFilePath string) error {
	return m.client.FGetObject(context.Background(), m.bucketName, objectName, localFilePath, minio.GetObjectOptions{})
}

func (m *MinioService) DeleteFile(objectName string) error {
	return m.client.RemoveObject(context.Background(), m.bucketName, objectName, minio.RemoveObjectOptions{})
}

func (m *MinioService) GetFileURL(objectName string) string {
	// In production, you might want to generate presigned URLs
	return fmt.Sprintf("/files/%s", objectName)
}

// GetContentType Helper function to determine content type
func GetContentType(extension string) string {
	switch extension {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

func (s *MinioService) DeleteObjectsByPrefix(prefix string) error {
	ctx := context.Background()
	log.Printf("[MinIO] Starting deletion for prefix: %s (bucket: %s)", prefix, s.bucketName)

	// Check bucket exists
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		log.Printf("[MinIO] Bucket check failed: %v", err)
		return err
	}
	if !exists {
		log.Printf("[MinIO] Bucket '%s' does not exist", s.bucketName)
		return nil // safe to skip
	}

	objectsCh := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// Debug: log if no objects
	objectCount := 0
	var objectKeys []string
	for obj := range objectsCh {
		if obj.Err != nil {
			log.Printf("[MinIO] List error: %v", obj.Err)
			return obj.Err
		}
		if obj.Key != "" {
			objectCount++
			objectKeys = append(objectKeys, obj.Key)
		}
	}

	if objectCount == 0 {
		log.Printf("[MinIO] No objects found with prefix: %s", prefix)
		return nil
	}

	log.Printf("[MinIO] Found %d objects to delete: %v", objectCount, objectKeys)

	errorCh := s.client.RemoveObjects(ctx, s.bucketName, s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}), minio.RemoveObjectsOptions{})

	for removeErr := range errorCh {
		if removeErr.Err != nil {
			log.Printf("[MinIO] Failed to delete object %s: %v", removeErr.ObjectName, removeErr.Err)
			return removeErr.Err
		}
	}

	log.Printf("[MinIO] Successfully deleted %d objects", objectCount)
	return nil
}
