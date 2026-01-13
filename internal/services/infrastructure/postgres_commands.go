package infrastructure

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/File-Sharing-BondBridg/File-Service/internal/models"
)

func (p *PostgresStorage) createTables() error {
	query := `
  CREATE TABLE IF NOT EXISTS files (
      id UUID PRIMARY KEY,
      name VARCHAR(255) NOT NULL,
      original_name VARCHAR(255) NOT NULL,
      size BIGINT NOT NULL,
      type VARCHAR(50) NOT NULL,
      extension VARCHAR(10) NOT NULL,
      uploaded_at TIMESTAMPTZ NOT NULL,
      file_path VARCHAR(500),
      preview_path VARCHAR(500),
      share_url VARCHAR(500),
      bucket_name VARCHAR(100) DEFAULT 'files',
      is_processed BOOLEAN DEFAULT false,
      scan_status VARCHAR(50) DEFAULT 'pending',
      scanned_at TIMESTAMPTZ,
      created_at TIMESTAMPTZ DEFAULT NOW(),
      updated_at TIMESTAMPTZ DEFAULT NOW(),
      user_id UUID NOT NULL
  );
  `
	_, err := p.Db.Exec(query)
	if err != nil {
		return err
	}

	// Idempotent: Add columns if missing (safe on restarts)
	alterQueries := []string{
		`ALTER TABLE files ADD COLUMN IF NOT EXISTS scan_status VARCHAR(50) DEFAULT 'pending'`,
		`ALTER TABLE files ADD COLUMN IF NOT EXISTS scanned_at TIMESTAMPTZ`,
		// Optional: Update existing rows to 'pending' if needed
		// `UPDATE files SET scan_status = 'pending' WHERE scan_status IS NULL`,
	}
	for _, altQuery := range alterQueries {
		_, err := p.Db.Exec(altQuery)
		if err != nil {
			// Log but don't fail—some DBs may error if column exists
			log.Printf("Warning during ALTER: %v", err)
		}
	}

	// Indexes
	indexQuery := `
  CREATE INDEX IF NOT EXISTS idx_files_uploaded_at ON files(uploaded_at DESC);
  CREATE INDEX IF NOT EXISTS idx_files_type ON files(type);
  CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
  CREATE INDEX IF NOT EXISTS idx_files_scan_status ON files(scan_status);
  `

	_, err = p.Db.Exec(indexQuery)
	return err
}

// Private methods with actual implementation
func (p *PostgresStorage) SaveFileMetadata(metadata models.FileMetadata) error {
	query := `
  INSERT INTO files (id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name, user_id, scan_status)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
  ON CONFLICT (id) DO UPDATE SET
      name = EXCLUDED.name,
      original_name = EXCLUDED.original_name,
      size = EXCLUDED.size,
      type = EXCLUDED.type,
      extension = EXCLUDED.extension,
      file_path = EXCLUDED.file_path,
      preview_path = EXCLUDED.preview_path,
      share_url = EXCLUDED.share_url,
      user_id = EXCLUDED.user_id,
      scan_status = EXCLUDED.scan_status,
      updated_at = NOW()
  `

	_, err := p.Db.Exec(query,
		metadata.ID,
		metadata.Name,
		metadata.OriginalName,
		metadata.Size,
		metadata.Type,
		metadata.Extension,
		metadata.UploadedAt,
		metadata.FilePath,
		metadata.PreviewPath,
		metadata.ShareURL,
		"files",
		metadata.UserID,
		"pending",
	)

	return err
}

func (p *PostgresStorage) GetFileMetadata(fileID string) (models.FileMetadata, bool) {
	query := `
  SELECT id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name, user_id, scan_status, scanned_at
  FROM files WHERE id = $1
  `

	var metadata models.FileMetadata
	err := p.Db.QueryRow(query, fileID).Scan(
		&metadata.ID,
		&metadata.Name,
		&metadata.OriginalName,
		&metadata.Size,
		&metadata.Type,
		&metadata.Extension,
		&metadata.UploadedAt,
		&metadata.FilePath,
		&metadata.PreviewPath,
		&metadata.ShareURL,
		&metadata.BucketName,
		&metadata.UserID,
		&metadata.ScanStatus,
		&metadata.ScannedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.FileMetadata{}, false
		}
		log.Printf("Error getting file metadata: %v", err)
		return models.FileMetadata{}, false
	}

	return metadata, true
}

func (p *PostgresStorage) getAllFileMetadataPerUser(userID string) []models.FileMetadata {
	query := `
  SELECT id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name, user_id, scan_status, scanned_at
  FROM files WHERE user_id = $1 ORDER BY uploaded_at DESC
  `

	rows, err := p.Db.Query(query, userID)
	if err != nil {
		log.Printf("Error querying all files: %v", err)
		return []models.FileMetadata{}
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}(rows)

	var files []models.FileMetadata
	for rows.Next() {
		var metadata models.FileMetadata
		err := rows.Scan(
			&metadata.ID,
			&metadata.Name,
			&metadata.OriginalName,
			&metadata.Size,
			&metadata.Type,
			&metadata.Extension,
			&metadata.UploadedAt,
			&metadata.FilePath,
			&metadata.PreviewPath,
			&metadata.ShareURL,
			&metadata.BucketName,
			&metadata.UserID,
			&metadata.ScanStatus,
			&metadata.ScannedAt,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		files = append(files, metadata)
	}

	return files
}

func (p *PostgresStorage) GetUserFileMetadata(userID string) []models.FileMetadata { // Renamed and added param
	query := `
      SELECT id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name, user_id  -- Added user_id
      FROM files WHERE user_id = $1 ORDER BY uploaded_at DESC  -- Added WHERE clause
  `
	rows, err := p.Db.Query(query, userID) // Pass userID
	if err != nil {
		log.Printf("Error querying user files: %v", err)
		return []models.FileMetadata{}
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}(rows)
	var files []models.FileMetadata
	for rows.Next() {
		var metadata models.FileMetadata
		err := rows.Scan(
			&metadata.ID,
			&metadata.Name,
			&metadata.OriginalName,
			&metadata.Size,
			&metadata.Type,
			&metadata.Extension,
			&metadata.UploadedAt,
			&metadata.FilePath,
			&metadata.PreviewPath,
			&metadata.ShareURL,
			&metadata.BucketName,
			&metadata.UserID,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		files = append(files, metadata)
	}
	return files
}

// GetUserFileMetadataPage returns a page of files for a user
func (p *PostgresStorage) GetUserFileMetadataPage(userID string, limit, offset int) ([]models.FileMetadata, error) {
	query := `
      SELECT id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name, user_id
      FROM files WHERE user_id = $1 ORDER BY uploaded_at DESC LIMIT $2 OFFSET $3
  `
	rows, err := p.Db.Query(query, userID, limit, offset)
	if err != nil {
		log.Printf("Error querying paginated user files: %v", err)
		return []models.FileMetadata{}, err
	}
	defer func(rows *sql.Rows) {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("Error closing rows: %v", cerr)
		}
	}(rows)
	var files []models.FileMetadata
	for rows.Next() {
		var metadata models.FileMetadata
		if err := rows.Scan(
			&metadata.ID,
			&metadata.Name,
			&metadata.OriginalName,
			&metadata.Size,
			&metadata.Type,
			&metadata.Extension,
			&metadata.UploadedAt,
			&metadata.FilePath,
			&metadata.PreviewPath,
			&metadata.ShareURL,
			&metadata.BucketName,
			&metadata.UserID,
		); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		files = append(files, metadata)
	}
	return files, nil
}

// GetUserFileCount counts total files for a user
func (p *PostgresStorage) GetUserFileCount(userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM files WHERE user_id = $1`
	var total int64
	err := p.Db.QueryRow(query, userID).Scan(&total)
	if err != nil {
		log.Printf("Error counting user files: %v", err)
		return 0, err
	}
	return total, nil
}

func (p *PostgresStorage) DeleteFileMetadata(fileID, userID string) bool {
	query := `DELETE FROM files WHERE id = $1 AND user_id = $2` // Added AND user_id
	result, err := p.Db.Exec(query, fileID, userID)
	if err != nil {
		log.Printf("Error deleting file metadata: %v", err)
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0
}

func (p *PostgresStorage) getStats() map[string]interface{} {
	var totalFiles int
	var totalSize int64
	var latestUpload time.Time

	err := p.Db.QueryRow(`
      SELECT COUNT(*), COALESCE(SUM(size), 0), COALESCE(MAX(uploaded_at), NOW())
      FROM files
  `).Scan(&totalFiles, &totalSize, &latestUpload)

	if err != nil {
		log.Printf("Error getting stats: %v", err)
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"total_files":   totalFiles,
		"total_size_mb": float64(totalSize) / (1024 * 1024),
		"latest_upload": latestUpload,
	}
}

func (p *PostgresStorage) DeleteAllFilesForUser(userID string) int {
	res, err := p.Db.Exec(`DELETE FROM files WHERE user_id = $1`, userID)
	if err != nil {
		log.Printf("Failed to delete files for user %s: %v", userID, err)
		return 0
	}
	count, _ := res.RowsAffected()
	return int(count)
}

func (p *PostgresStorage) UpdateFileScanStatus(
	fileID, status string,
	scannedAt time.Time,
) error {
	query := `
      UPDATE files
      SET scan_status = $1,
          scanned_at = $2,
          updated_at = NOW()
      WHERE id = $3
  `
	_, err := p.Db.Exec(query, status, scannedAt, fileID)
	return err
}
