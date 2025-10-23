package storage

import (
    "encoding/json"
    "fmt"
    "os"
    "sort"
    "sync"

    "github.com/File-Sharing-BondBridg/File-Service/internal/models"
)

// LocalStorage implements Storage interface for local JSON file storage
type LocalStorage struct {
    fileMetadataStore map[string]models.FileMetadata
    metadataMutex     sync.RWMutex
}

func (l *LocalStorage) SaveFileMetadata(metadata models.FileMetadata) error {
    l.metadataMutex.Lock()
    l.fileMetadataStore[metadata.ID] = metadata
    l.metadataMutex.Unlock()

    return l.saveToFile()
}

func (l *LocalStorage) GetFileMetadata(fileID string) (models.FileMetadata, bool) {
    l.metadataMutex.RLock()
    defer l.metadataMutex.RUnlock()
    metadata, exists := l.fileMetadataStore[fileID]
    return metadata, exists
}

func (l *LocalStorage) GetAllFileMetadata() []models.FileMetadata {
    l.metadataMutex.RLock()
    defer l.metadataMutex.RUnlock()

    files := make([]models.FileMetadata, 0, len(l.fileMetadataStore))
    for _, metadata := range l.fileMetadataStore {
        files = append(files, metadata)
    }

    sort.Slice(files, func(i, j int) bool {
        return files[i].UploadedAt.After(files[j].UploadedAt)
    })

    return files
}

func (l *LocalStorage) DeleteFileMetadata(fileID string) bool {
    l.metadataMutex.Lock()
    defer l.metadataMutex.Unlock()

    if _, exists := l.fileMetadataStore[fileID]; exists {
        delete(l.fileMetadataStore, fileID)
        return l.saveToFile() == nil
    }
    return false
}

const metadataFile = "file_metadata.json"

// In-memory storage
var fileMetadataStore = make(map[string]models.FileMetadata)
var metadataMutex sync.RWMutex

// Initialize loads metadata from JSON file on startup
func Initialize() error {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	// Check if metadata file exists
	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		// File doesn't exist, start with empty store
		return nil
	}

	// Read the file
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %v", err)
	}

	// Unmarshal JSON data
	if err := json.Unmarshal(data, &fileMetadataStore); err != nil {
		return fmt.Errorf("failed to parse metadata file: %v", err)
	}

	fmt.Printf("Loaded %d file metadata entries from %s\n", len(fileMetadataStore), metadataFile)
	return nil
}

// saveToFile writes the current metadata to JSON file
func saveToFile() error {
	//metadataMutex.RLock()
	//defer metadataMutex.RUnlock()

	data, err := json.MarshalIndent(fileMetadataStore, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	// Write to temporary file first for atomicity
	tempFile := metadataFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %v", err)
	}

	// Rename temp file to actual file (atomic operation)
	if err := os.Rename(tempFile, metadataFile); err != nil {
		return fmt.Errorf("failed to rename metadata file: %v", err)
	}

	return nil
}

func SaveFileMetadata(metadata models.FileMetadata) error {
	metadataMutex.Lock()
	fileMetadataStore[metadata.ID] = metadata
	metadataMutex.Unlock()

	// Persist to disk
	if err := saveToFile(); err != nil {
		// If save fails, remove from memory to maintain consistency
		metadataMutex.Lock()
		delete(fileMetadataStore, metadata.ID)
		metadataMutex.Unlock()
		return fmt.Errorf("failed to persist metadata: %v", err)
	}

	return nil
}

func GetFileMetadata(fileID string) (models.FileMetadata, bool) {
	metadataMutex.RLock()
	defer metadataMutex.RUnlock()
	metadata, exists := fileMetadataStore[fileID]
	return metadata, exists
}

func GetAllFileMetadata() []models.FileMetadata {
	metadataMutex.RLock()
	defer metadataMutex.RUnlock()

	files := make([]models.FileMetadata, 0, len(fileMetadataStore))
	for _, metadata := range fileMetadataStore {
		files = append(files, metadata)
	}

	// Sort by upload date (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].UploadedAt.After(files[j].UploadedAt)
	})

	return files
}

func DeleteFileMetadata(fileID string) bool {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	if _, exists := fileMetadataStore[fileID]; exists {
		delete(fileMetadataStore, fileID)

		// Persist changes to disk
		if err := saveToFile(); err != nil {
			fmt.Printf("Warning: Failed to persist metadata deletion: %v\n", err)
			// Don't return error here as the in-memory delete was successful
		}
		return true
	}
	return false
}

// GetStats returns storage statistics (useful for debugging)
func GetStats() map[string]interface{} {
	metadataMutex.RLock()
	defer metadataMutex.RUnlock()

	return map[string]interface{}{
		"total_files": len(fileMetadataStore),
		"file_ids":    getFileIDs(),
	}
}

// getFileIDs returns all file IDs (for debugging)
func getFileIDs() []string {
	ids := make([]string, 0, len(fileMetadataStore))
	for id := range fileMetadataStore {
		ids = append(ids, id)
	}
	return ids
}
