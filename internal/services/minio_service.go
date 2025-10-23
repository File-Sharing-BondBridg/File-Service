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
        Creds:  credentials.New(accessKey, secretKey),
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

// Helper function to determine content type
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