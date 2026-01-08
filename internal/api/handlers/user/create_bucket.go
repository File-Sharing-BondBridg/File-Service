package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	"github.com/minio/minio-go/v7"
)

func HandleUserSynced(ctx context.Context, data []byte) error {
	var event struct {
		EventType string `json:"eventType"`
		UserID    string `json:"userId"`
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	if event.EventType != "UserSynced" {
		return nil
	}

	minioSvc := services.GetMinioService()
	if minioSvc == nil {
		return errors.New("minio not initialized")
	}

	objectName := fmt.Sprintf("users/%s/.init", event.UserID)

	_, err := minioSvc.Client.PutObject(
		ctx,
		minioSvc.BucketName,
		objectName,
		strings.NewReader(""),
		0,
		minio.PutObjectOptions{},
	)

	return err
}
