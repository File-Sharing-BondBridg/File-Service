package command

import (
	"time"

	"github.com/File-Sharing-BondBridg/File-Service/internal/models"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services/infrastructure"
)

var postgresInstance *infrastructure.PostgresStorage

func SaveFileMetadata(metadata models.FileMetadata) error {
	// Default implementation
	//if postgresInstance == nil {
	//	return fmt.Errorf("postgres storage not initialized")
	//}
	//return postgresInstance.saveFileMetadata(metadata)

	// Sharding implementation
	pg := infrastructure.GetPostgresForUser(metadata.UserID)
	return pg.SaveFileMetadata(metadata)
}

func DeleteFileMetadata(fileID, userID string) bool {
	pg := infrastructure.GetPostgresForUser(userID)
	return pg.DeleteFileMetadata(fileID, userID)
}

func UpdateFileScanStatus(fileID, userID, status string, now time.Time) error {
	pg := infrastructure.GetPostgresForUser(userID)
	return pg.UpdateFileScanStatus(fileID, status, now)
}

func DeleteAllFilesForUser(userID string) int {
	pg := infrastructure.GetPostgresForUser(userID)
	return pg.DeleteAllFilesForUser(userID)
}
