package storage

import "github.com/File-Sharing-BondBridg/File-Service/internal/models"

// Storage interface defines the contract for all storage implementations
type Storage interface {
    SaveFileMetadata(metadata models.FileMetadata) error
    GetFileMetadata(fileID string) (models.FileMetadata, bool)
    GetAllFileMetadata() []models.FileMetadata
    DeleteFileMetadata(fileID string) bool
}

// Global storage instance
var currentStorage Storage = &LocalStorage{}

// Initialize sets up the storage system (currently using local storage as default)
func Initialize() error {
    // For now, use local storage as default
    // This maintains backward compatibility with your existing code
    return nil
}

// InitializePostgres sets up PostgreSQL storage
func InitializePostgres(connectionString string) error {
    pgStorage := &PostgresStorage{}
    if err := pgStorage.Connect(connectionString); err != nil {
        return err
    }
    currentStorage = pgStorage
    return nil
}

// Public functions that use the current storage implementation
func SaveFileMetadata(metadata models.FileMetadata) error {
    return currentStorage.SaveFileMetadata(metadata)
}

func GetFileMetadata(fileID string) (models.FileMetadata, bool) {
    return currentStorage.GetFileMetadata(fileID)
}

func GetAllFileMetadata() []models.FileMetadata {
    return currentStorage.GetAllFileMetadata()
}

func DeleteFileMetadata(fileID string) bool {
    return currentStorage.DeleteFileMetadata(fileID)
}