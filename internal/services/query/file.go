package query

import (
	"fmt"

	"github.com/File-Sharing-BondBridg/File-Service/internal/models"
	"github.com/File-Sharing-BondBridg/File-Service/internal/services/infrastructure"
)

var postgresInstance *infrastructure.PostgresStorage

// GetUserFileMetadataPage returns a paginated list of files for a user
func GetUserFileMetadataPage(userID string, limit, offset int) ([]models.FileMetadata, error) {
	pg := infrastructure.GetPostgresForUser(userID)
	return pg.GetUserFileMetadataPage(userID, limit, offset)
}

// GetUserFileCount returns the total number of files for a user
func GetUserFileCount(userID string) (int64, error) {
	pg := infrastructure.GetPostgresForUser(userID)
	return pg.GetUserFileCount(userID)
}

func GetUserFileStats(userID string) (models.UserFileStats, error) {
	pg := infrastructure.GetPostgresForUser(userID)
	if pg == nil {
		return models.UserFileStats{}, nil
	}

	count, err := pg.GetUserFileStats(userID)
	if err != nil {
		return models.UserFileStats{}, err
	}

	return models.UserFileStats{
		FileCount: count,
	}, nil
}

// GetFileMetadataForUser retrieves metadata of a file associated with a specific user based on the provided fileID and userID.
func GetFileMetadataForUser(fileID, userID string) (models.FileMetadata, bool) {
	pg := infrastructure.GetPostgresForUser(userID)
	return pg.GetFileMetadata(fileID)
}

func GetFileMetadata(fileID string) (models.FileMetadata, bool) {
	if postgresInstance == nil {
		return models.FileMetadata{}, false
	}
	return postgresInstance.GetFileMetadata(fileID)
}

func GetFilePathsForUser(userID string) ([]string, error) {
	pg := infrastructure.GetPostgresForUser(userID)
	if pg == nil || pg.Db == nil {
		return nil, fmt.Errorf("postgres shard not initialized for user %s", userID)
	}

	var paths []string
	rows, err := pg.Db.Query(
		`SELECT file_path FROM files WHERE user_id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	return paths, rows.Err()
}
