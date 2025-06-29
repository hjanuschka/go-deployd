package storage

import (
	"context"
	"fmt"
	"io"
	
	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/logging"
)

// Manager handles file storage operations
type Manager struct {
	storage StorageInterface
	config  *config.StorageConfig
}

// NewManager creates a new storage manager based on configuration
func NewManager(cfg *config.StorageConfig, db database.DatabaseInterface, baseURL string) (*Manager, error) {
	var storage StorageInterface
	var err error
	
	switch cfg.Type {
	case "local":
		storage, err = NewLocalStorage(cfg, db, baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create local storage: %w", err)
		}
		logging.Info("Initialized local file storage", "storage", map[string]interface{}{
			"basePath":  cfg.Local.BasePath,
			"urlPrefix": cfg.Local.URLPrefix,
		})
		
	case "s3", "minio":
		storage, err = NewS3Storage(cfg, db, baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create S3/MinIO storage: %w", err)
		}
		logging.Info("Initialized S3/MinIO file storage", "storage", map[string]interface{}{
			"type":     cfg.Type,
			"bucket":   cfg.S3.Bucket,
			"endpoint": cfg.S3.Endpoint,
		})
		
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
	
	return &Manager{
		storage: storage,
		config:  cfg,
	}, nil
}

// GetStorage returns the underlying storage interface
func (m *Manager) GetStorage() StorageInterface {
	return m.storage
}

// GetConfig returns the storage configuration
func (m *Manager) GetConfig() *config.StorageConfig {
	return m.config
}

// ValidateFile checks if a file meets the configured constraints
func (m *Manager) ValidateFile(filename string, size int64) error {
	// Check file size
	if size > m.config.MaxFileSize {
		return FileTooLargeError{
			Size:    size,
			MaxSize: m.config.MaxFileSize,
		}
	}
	
	// Check file extension
	if len(m.config.AllowedExtensions) > 0 {
		ext := getFileExtension(filename)
		if !isExtensionAllowed(ext, m.config.AllowedExtensions) {
			return InvalidFileTypeError{Extension: ext}
		}
	}
	
	return nil
}

// Convenience methods that delegate to the storage interface

func (m *Manager) Upload(ctx context.Context, filename string, reader io.Reader, options *UploadOptions) (*FileInfo, error) {
	return m.storage.Upload(ctx, filename, reader, options)
}

func (m *Manager) Download(ctx context.Context, fileID string) (io.ReadCloser, *FileInfo, error) {
	return m.storage.Download(ctx, fileID)
}

func (m *Manager) Delete(ctx context.Context, fileID string) error {
	return m.storage.Delete(ctx, fileID)
}

func (m *Manager) GetInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	return m.storage.GetInfo(ctx, fileID)
}

func (m *Manager) GenerateSignedURL(ctx context.Context, fileID string, options *SignedURLOptions) (string, error) {
	return m.storage.GenerateSignedURL(ctx, fileID, options)
}

func (m *Manager) List(ctx context.Context, options ListOptions) ([]*FileInfo, error) {
	return m.storage.List(ctx, options)
}

// Helper function
func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}