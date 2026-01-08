package services

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	Client     *minio.Client
	BucketName string
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
		Client:     client,
		BucketName: bucket,
	}

	log.Println("Connected to MinIO successfully")
	return nil
}

func GetMinioService() *MinioService {
	return minioInstance
}

// CheckConnection Add this method for health checks
func (m *MinioService) CheckConnection() error {
	if m == nil || m.Client == nil {
		return fmt.Errorf("minio service not initialized")
	}
	_, err := m.Client.BucketExists(context.Background(), m.BucketName)
	return err
}

func (s *MinioService) UploadFile(reader io.Reader, size int64, objectName, contentType string) error {
	ctx := context.Background()

	_, err := s.Client.PutObject(
		ctx,
		s.BucketName,
		objectName,
		reader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	return err
}

func (m *MinioService) DownloadFile(objectName, localFilePath string) error {
	return m.Client.FGetObject(context.Background(), m.BucketName, objectName, localFilePath, minio.GetObjectOptions{})
}

func (m *MinioService) DeleteFile(objectName string) error {
	return m.Client.RemoveObject(context.Background(), m.BucketName, objectName, minio.RemoveObjectOptions{})
}

func (m *MinioService) GetFileURL(objectName string) string {
	// In production, you might want to generate presigned URLs
	return fmt.Sprintf("/files/%s", objectName)
}

// GetContentType Helper function to determine the content type
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
	log.Printf("[MinIO] Starting deletion for prefix: %s (bucket: %s)", prefix, s.BucketName)

	// Check bucket exists
	exists, err := s.Client.BucketExists(ctx, s.BucketName)
	if err != nil {
		log.Printf("[MinIO] Bucket check failed: %v", err)
		return err
	}
	if !exists {
		log.Printf("[MinIO] Bucket '%s' does not exist", s.BucketName)
		return nil // safe to skip
	}

	objectsCh := s.Client.ListObjects(ctx, s.BucketName, minio.ListObjectsOptions{
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

	errorCh := s.Client.RemoveObjects(ctx, s.BucketName, s.Client.ListObjects(ctx, s.BucketName, minio.ListObjectsOptions{
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
