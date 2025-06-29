package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/logging"
)

// LocalStorage implements StorageInterface for local file storage
type LocalStorage struct {
	config *config.StorageConfig
	db     database.DatabaseInterface
	baseURL string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(cfg *config.StorageConfig, db database.DatabaseInterface, baseURL string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(cfg.Local.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return &LocalStorage{
		config:  cfg,
		db:      db,
		baseURL: baseURL,
	}, nil
}

// Upload stores a file locally
func (ls *LocalStorage) Upload(ctx context.Context, filename string, reader io.Reader, options *UploadOptions) (*FileInfo, error) {
	// Generate unique file ID
	fileID := generateFileID()
	
	// Sanitize filename
	safeFilename := sanitizeFilename(filename)
	ext := filepath.Ext(safeFilename)
	
	// Check allowed extensions
	if len(ls.config.AllowedExtensions) > 0 && !isExtensionAllowed(ext, ls.config.AllowedExtensions) {
		return nil, InvalidFileTypeError{Extension: ext}
	}
	
	// Create file path
	datePath := time.Now().Format("2006/01/02")
	dirPath := filepath.Join(ls.config.Local.BasePath, datePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	storedFilename := fmt.Sprintf("%s%s", fileID, ext)
	filePath := filepath.Join(dirPath, storedFilename)
	
	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	// Copy content with size check
	var size int64
	limitReader := &limitedReader{
		reader:  reader,
		maxSize: ls.config.MaxFileSize,
	}
	
	size, err = io.Copy(file, limitReader)
	if err != nil {
		os.Remove(filePath) // Clean up on error
		if err == errFileTooLarge {
			return nil, FileTooLargeError{Size: limitReader.bytesRead, MaxSize: ls.config.MaxFileSize}
		}
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	
	// Create file info
	fileInfo := &FileInfo{
		ID:           fileID,
		Filename:     storedFilename,
		OriginalName: filename,
		ContentType:  options.ContentType,
		Size:         size,
		StorageType:  "local",
		Path:         filePath,
		URL:          fmt.Sprintf("%s%s/%s", ls.baseURL, ls.config.Local.URLPrefix, fileID),
		UploadedAt:   time.Now(),
		UploadedBy:   options.UserID,
		Metadata:     options.Metadata,
	}
	
	// Store metadata in database
	if err := ls.storeMetadata(ctx, fileInfo); err != nil {
		os.Remove(filePath) // Clean up on error
		return nil, fmt.Errorf("failed to store metadata: %w", err)
	}
	
	logging.Info("File uploaded successfully", "storage", map[string]interface{}{
		"fileID":   fileID,
		"filename": filename,
		"size":     size,
	})
	
	return fileInfo, nil
}

// Download retrieves a file from local storage
func (ls *LocalStorage) Download(ctx context.Context, fileID string) (io.ReadCloser, *FileInfo, error) {
	// Get metadata from database
	fileInfo, err := ls.getMetadata(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	
	// Open file
	file, err := os.Open(fileInfo.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, FileNotFoundError{FileID: fileID}
		}
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return file, fileInfo, nil
}

// Delete removes a file from local storage
func (ls *LocalStorage) Delete(ctx context.Context, fileID string) error {
	// Get metadata
	fileInfo, err := ls.getMetadata(ctx, fileID)
	if err != nil {
		return err
	}
	
	// Delete file
	if err := os.Remove(fileInfo.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	// Delete metadata
	if err := ls.deleteMetadata(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}
	
	logging.Info("File deleted successfully", "storage", map[string]interface{}{
		"fileID": fileID,
	})
	
	return nil
}

// GetInfo retrieves file metadata
func (ls *LocalStorage) GetInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	return ls.getMetadata(ctx, fileID)
}

// GenerateSignedURL generates a signed URL for local storage
func (ls *LocalStorage) GenerateSignedURL(ctx context.Context, fileID string, options *SignedURLOptions) (string, error) {
	if options.Operation == "put" {
		// For local storage, we'll return a special upload URL
		// This will be handled by our upload endpoint
		return fmt.Sprintf("%s%s/upload/%s", ls.baseURL, ls.config.Local.URLPrefix, fileID), nil
	}
	
	// For GET operations, just return the regular URL
	return fmt.Sprintf("%s%s/%s", ls.baseURL, ls.config.Local.URLPrefix, fileID), nil
}

// List files with filtering
func (ls *LocalStorage) List(ctx context.Context, options ListOptions) ([]*FileInfo, error) {
	// Build query
	store := ls.db.CreateStore("files")
	query := database.NewQueryBuilder()
	
	if options.UserID != "" {
		query.Where("uploadedBy", "$eq", options.UserID)
	}
	if options.AfterDate != nil {
		query.Where("uploadedAt", "$gte", options.AfterDate.Format(time.RFC3339))
	}
	if options.BeforeDate != nil {
		query.Where("uploadedAt", "$lte", options.BeforeDate.Format(time.RFC3339))
	}
	
	// Query options for sorting and pagination
	limit := int64(options.Limit)
	offset := int64(options.Offset)
	queryOpts := database.QueryOptions{
		Sort:  map[string]int{"uploadedAt": -1}, // Newest first
		Limit: &limit,
		Skip:  &offset,
	}
	
	// Execute query
	results, err := store.Find(ctx, query, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	// Convert to FileInfo
	files := make([]*FileInfo, 0, len(results))
	for _, result := range results {
		fileInfo := &FileInfo{}
		if err := mapToFileInfo(result, fileInfo); err != nil {
			logging.Error("Failed to parse file info", "storage", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}
		files = append(files, fileInfo)
	}
	
	return files, nil
}

// Helper methods

func (ls *LocalStorage) storeMetadata(ctx context.Context, fileInfo *FileInfo) error {
	store := ls.db.CreateStore("files")
	
	data := map[string]interface{}{
		"id":           fileInfo.ID,
		"filename":     fileInfo.Filename,
		"originalName": fileInfo.OriginalName,
		"contentType":  fileInfo.ContentType,
		"size":         fileInfo.Size,
		"storageType":  fileInfo.StorageType,
		"path":         fileInfo.Path,
		"url":          fileInfo.URL,
		"uploadedAt":   fileInfo.UploadedAt,
		"uploadedBy":   fileInfo.UploadedBy,
		"metadata":     fileInfo.Metadata,
	}
	
	_, err := store.Insert(ctx, data)
	return err
}

func (ls *LocalStorage) getMetadata(ctx context.Context, fileID string) (*FileInfo, error) {
	store := ls.db.CreateStore("files")
	
	query := database.NewQueryBuilder().Where("id", "$eq", fileID)
	result, err := store.FindOne(ctx, query)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, FileNotFoundError{FileID: fileID}
	}
	
	fileInfo := &FileInfo{}
	if err := mapToFileInfo(result, fileInfo); err != nil {
		return nil, err
	}
	
	return fileInfo, nil
}

func (ls *LocalStorage) deleteMetadata(ctx context.Context, fileID string) error {
	store := ls.db.CreateStore("files")
	query := database.NewQueryBuilder().Where("id", "$eq", fileID)
	_, err := store.Remove(ctx, query)
	return err
}

// Utility functions

func generateFileID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func sanitizeFilename(filename string) string {
	// Remove path separators and other dangerous characters
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, "..", "")
	
	// Replace spaces with underscores
	filename = strings.ReplaceAll(filename, " ", "_")
	
	// Remove any remaining special characters except dots and hyphens
	safe := strings.Builder{}
	for _, r := range filename {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
		   (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			safe.WriteRune(r)
		}
	}
	
	return safe.String()
}

func isExtensionAllowed(ext string, allowed []string) bool {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, a := range allowed {
		if strings.ToLower(strings.TrimPrefix(a, ".")) == ext {
			return true
		}
	}
	return false
}

func mapToFileInfo(data map[string]interface{}, info *FileInfo) error {
	if id, ok := data["id"].(string); ok {
		info.ID = id
	}
	if filename, ok := data["filename"].(string); ok {
		info.Filename = filename
	}
	if originalName, ok := data["originalName"].(string); ok {
		info.OriginalName = originalName
	}
	if contentType, ok := data["contentType"].(string); ok {
		info.ContentType = contentType
	}
	if size, ok := data["size"].(float64); ok {
		info.Size = int64(size)
	}
	if storageType, ok := data["storageType"].(string); ok {
		info.StorageType = storageType
	}
	if path, ok := data["path"].(string); ok {
		info.Path = path
	}
	if url, ok := data["url"].(string); ok {
		info.URL = url
	}
	if uploadedBy, ok := data["uploadedBy"].(string); ok {
		info.UploadedBy = uploadedBy
	}
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		info.Metadata = metadata
	}
	
	// Parse time
	if uploadedAt, ok := data["uploadedAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, uploadedAt); err == nil {
			info.UploadedAt = t
		}
	}
	
	return nil
}

// limitedReader limits the amount of data read
type limitedReader struct {
	reader    io.Reader
	maxSize   int64
	bytesRead int64
}

var errFileTooLarge = fmt.Errorf("file too large")

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	n, err = lr.reader.Read(p)
	lr.bytesRead += int64(n)
	
	if lr.bytesRead > lr.maxSize {
		return n, errFileTooLarge
	}
	
	return n, err
}