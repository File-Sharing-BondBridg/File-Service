package storage

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "time"

    "github.com/File-Sharing-BondBridg/File-Service/internal/models"
    _ "github.com/lib/pq"
)

// PostgresStorage implements Storage interface for PostgreSQL
type PostgresStorage struct {
    db *sql.DB
}

// Connect establishes connection to PostgreSQL
func (p *PostgresStorage) Connect(connectionString string) error {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
    }

    // Test the connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("failed to ping PostgreSQL: %v", err)
    }

    // Set connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)

    p.db = db

    // Create tables
    if err := p.createTables(); err != nil {
        return fmt.Errorf("failed to create tables: %v", err)
    }

    log.Println("✅ Connected to PostgreSQL successfully")
    return nil
}

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
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    );

    CREATE INDEX IF NOT EXISTS idx_files_uploaded_at ON files(uploaded_at DESC);
    CREATE INDEX IF NOT EXISTS idx_files_type ON files(type);
    `

    _, err := p.db.Exec(query)
    return err
}

// Implement Storage interface methods
func (p *PostgresStorage) SaveFileMetadata(metadata models.FileMetadata) error {
    query := `
    INSERT INTO files (id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        original_name = EXCLUDED.original_name,
        size = EXCLUDED.size,
        type = EXCLUDED.type,
        extension = EXCLUDED.extension,
        file_path = EXCLUDED.file_path,
        preview_path = EXCLUDED.preview_path,
        share_url = EXCLUDED.share_url,
        updated_at = NOW()
    `

    _, err := p.db.Exec(query,
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
    )

    return err
}

func (p *PostgresStorage) GetFileMetadata(fileID string) (models.FileMetadata, bool) {
    query := `
    SELECT id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name
    FROM files WHERE id = $1
    `

    var metadata models.FileMetadata
    err := p.db.QueryRow(query, fileID).Scan(
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
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return models.FileMetadata{}, false
        }
        log.Printf("Error getting file metadata: %v", err)
        return models.FileMetadata{}, false
    }

    return metadata, true
}

func (p *PostgresStorage) GetAllFileMetadata() []models.FileMetadata {
    query := `
    SELECT id, name, original_name, size, type, extension, uploaded_at, file_path, preview_path, share_url, bucket_name
    FROM files ORDER BY uploaded_at DESC
    `

    rows, err := p.db.Query(query)
    if err != nil {
        log.Printf("Error querying all files: %v", err)
        return []models.FileMetadata{}
    }
    defer rows.Close()

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
        )
        if err != nil {
            log.Printf("Error scanning row: %v", err)
            continue
        }
        files = append(files, metadata)
    }

    return files
}

func (p *PostgresStorage) DeleteFileMetadata(fileID string) bool {
    query := `DELETE FROM files WHERE id = $1`
    result, err := p.db.Exec(query, fileID)
    if err != nil {
        log.Printf("Error deleting file metadata: %v", err)
        return false
    }

    rowsAffected, _ := result.RowsAffected()
    return rowsAffected > 0
}