package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo represents metadata about a stored file
type FileInfo struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	Size         int64     `json:"size"`
	StorageType  string    `json:"storageType"`
	Path         string    `json:"path"`
	URL          string    `json:"url,omitempty"`
	UploadedAt   time.Time `json:"uploadedAt"`
	UploadedBy   string    `json:"uploadedBy,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UploadOptions for file uploads
type UploadOptions struct {
	ContentType string
	Metadata    map[string]interface{}
	UserID      string
}

// SignedURLOptions for generating signed URLs
type SignedURLOptions struct {
	Operation   string        // "get" or "put"
	ContentType string        // For PUT operations
	Expiration  time.Duration // URL expiration time
}

// StorageInterface defines the interface for file storage backends
type StorageInterface interface {
	// Upload stores a file and returns its metadata
	Upload(ctx context.Context, filename string, reader io.Reader, options *UploadOptions) (*FileInfo, error)
	
	// Download retrieves a file
	Download(ctx context.Context, fileID string) (io.ReadCloser, *FileInfo, error)
	
	// Delete removes a file
	Delete(ctx context.Context, fileID string) error
	
	// GetInfo retrieves file metadata without downloading
	GetInfo(ctx context.Context, fileID string) (*FileInfo, error)
	
	// GenerateSignedURL creates a signed URL for direct upload/download
	GenerateSignedURL(ctx context.Context, fileID string, options *SignedURLOptions) (string, error)
	
	// List files with optional filtering
	List(ctx context.Context, options ListOptions) ([]*FileInfo, error)
}

// ListOptions for listing files
type ListOptions struct {
	Limit      int
	Offset     int
	UserID     string
	AfterDate  *time.Time
	BeforeDate *time.Time
}

// Error types
type FileNotFoundError struct {
	FileID string
}

func (e FileNotFoundError) Error() string {
	return "file not found: " + e.FileID
}

type FileTooLargeError struct {
	Size    int64
	MaxSize int64
}

func (e FileTooLargeError) Error() string {
	return "file too large"
}

type InvalidFileTypeError struct {
	Extension string
}

func (e InvalidFileTypeError) Error() string {
	return "invalid file type: " + e.Extension
}