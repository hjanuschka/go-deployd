package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	
	appconfig "github.com/hjanuschka/go-deployd/internal/config"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/logging"
)

// S3Storage implements StorageInterface for S3/MinIO storage
type S3Storage struct {
	config  *appconfig.StorageConfig
	db      database.DatabaseInterface
	client  *s3.Client
	baseURL string
}

// NewS3Storage creates a new S3/MinIO storage instance
func NewS3Storage(cfg *appconfig.StorageConfig, db database.DatabaseInterface, baseURL string) (*S3Storage, error) {
	// Create custom endpoint resolver for MinIO
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.S3.Endpoint != "" {
			return aws.Endpoint{
				URL:               cfg.S3.Endpoint,
				SigningRegion:     cfg.S3.Region,
				HostnameImmutable: true,
			}, nil
		}
		// Use default resolver for AWS S3
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	
	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.S3.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3.AccessKeyID,
			cfg.S3.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	// Create S3 client with options
	clientOptions := func(o *s3.Options) {
		o.UsePathStyle = cfg.S3.PathStyle
		if cfg.Type == "minio" && !cfg.S3.UseSSL {
			o.EndpointOptions.DisableHTTPS = true
		}
	}
	
	client := s3.NewFromConfig(awsCfg, clientOptions)
	
	// Test bucket access
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(cfg.S3.Bucket),
	})
	if err != nil {
		// Try to create bucket if it doesn't exist
		_, createErr := client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(cfg.S3.Bucket),
		})
		if createErr != nil {
			return nil, fmt.Errorf("bucket not accessible and cannot create: %w", err)
		}
		logging.Info("Created S3 bucket", "storage", map[string]interface{}{
			"bucket": cfg.S3.Bucket,
		})
	}
	
	return &S3Storage{
		config:  cfg,
		db:      db,
		client:  client,
		baseURL: baseURL,
	}, nil
}

// Upload stores a file in S3/MinIO
func (s3s *S3Storage) Upload(ctx context.Context, filename string, reader io.Reader, options *UploadOptions) (*FileInfo, error) {
	// Generate unique file ID
	fileID := generateFileID()
	
	// Sanitize filename
	safeFilename := sanitizeFilename(filename)
	ext := filepath.Ext(safeFilename)
	
	// Check allowed extensions
	if len(s3s.config.AllowedExtensions) > 0 && !isExtensionAllowed(ext, s3s.config.AllowedExtensions) {
		return nil, InvalidFileTypeError{Extension: ext}
	}
	
	// Create S3 key with date-based path
	datePath := time.Now().Format("2006/01/02")
	storedFilename := fmt.Sprintf("%s%s", fileID, ext)
	s3Key := fmt.Sprintf("%s/%s", datePath, storedFilename)
	
	// Read content into buffer for size check
	buf := new(bytes.Buffer)
	limitReader := &limitedReader{
		reader:  reader,
		maxSize: s3s.config.MaxFileSize,
	}
	
	size, err := io.Copy(buf, limitReader)
	if err != nil {
		if err == errFileTooLarge {
			return nil, FileTooLargeError{Size: limitReader.bytesRead, MaxSize: s3s.config.MaxFileSize}
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	// Prepare metadata
	metadata := make(map[string]string)
	metadata["original-name"] = filename
	if options.UserID != "" {
		metadata["uploaded-by"] = options.UserID
	}
	
	// Upload to S3
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(s3s.config.S3.Bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(options.ContentType),
		Metadata:    metadata,
	}
	
	_, err = s3s.client.PutObject(ctx, putInput)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}
	
	// Generate URL
	url := s3s.generateURL(s3Key)
	
	// Create file info
	fileInfo := &FileInfo{
		ID:           fileID,
		Filename:     storedFilename,
		OriginalName: filename,
		ContentType:  options.ContentType,
		Size:         size,
		StorageType:  s3s.config.Type,
		Path:         s3Key,
		URL:          url,
		UploadedAt:   time.Now(),
		UploadedBy:   options.UserID,
		Metadata:     options.Metadata,
	}
	
	// Store metadata in database
	if err := s3s.storeMetadata(ctx, fileInfo); err != nil {
		// Try to clean up S3 object
		s3s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: aws.String(s3s.config.S3.Bucket),
			Key:    aws.String(s3Key),
		})
		return nil, fmt.Errorf("failed to store metadata: %w", err)
	}
	
	logging.Info("File uploaded to S3", "storage", map[string]interface{}{
		"fileID":   fileID,
		"filename": filename,
		"size":     size,
		"s3Key":    s3Key,
	})
	
	return fileInfo, nil
}

// Download retrieves a file from S3/MinIO
func (s3s *S3Storage) Download(ctx context.Context, fileID string) (io.ReadCloser, *FileInfo, error) {
	// Get metadata from database
	fileInfo, err := s3s.getMetadata(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	
	// Download from S3
	getInput := &s3.GetObjectInput{
		Bucket: aws.String(s3s.config.S3.Bucket),
		Key:    aws.String(fileInfo.Path),
	}
	
	result, err := s3s.client.GetObject(ctx, getInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	
	return result.Body, fileInfo, nil
}

// Delete removes a file from S3/MinIO
func (s3s *S3Storage) Delete(ctx context.Context, fileID string) error {
	// Get metadata
	fileInfo, err := s3s.getMetadata(ctx, fileID)
	if err != nil {
		return err
	}
	
	// Delete from S3
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(s3s.config.S3.Bucket),
		Key:    aws.String(fileInfo.Path),
	}
	
	_, err = s3s.client.DeleteObject(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	
	// Delete metadata
	if err := s3s.deleteMetadata(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}
	
	logging.Info("File deleted from S3", "storage", map[string]interface{}{
		"fileID": fileID,
		"s3Key":  fileInfo.Path,
	})
	
	return nil
}

// GetInfo retrieves file metadata
func (s3s *S3Storage) GetInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	return s3s.getMetadata(ctx, fileID)
}

// GenerateSignedURL generates a pre-signed URL for S3/MinIO
func (s3s *S3Storage) GenerateSignedURL(ctx context.Context, fileID string, options *SignedURLOptions) (string, error) {
	// For PUT operations, generate a new file ID
	var s3Key string
	if options.Operation == "put" {
		datePath := time.Now().Format("2006/01/02")
		s3Key = fmt.Sprintf("%s/%s", datePath, fileID)
	} else {
		// For GET operations, get the actual S3 key from metadata
		fileInfo, err := s3s.getMetadata(ctx, fileID)
		if err != nil {
			return "", err
		}
		s3Key = fileInfo.Path
	}
	
	// Create presign client
	presignClient := s3.NewPresignClient(s3s.client)
	
	// Set expiration
	expiration := time.Duration(s3s.config.SignedURLExpiration) * time.Second
	if options.Expiration > 0 {
		expiration = options.Expiration
	}
	
	if options.Operation == "put" {
		// Generate PUT presigned URL
		putInput := &s3.PutObjectInput{
			Bucket: aws.String(s3s.config.S3.Bucket),
			Key:    aws.String(s3Key),
		}
		if options.ContentType != "" {
			putInput.ContentType = aws.String(options.ContentType)
		}
		
		presignResult, err := presignClient.PresignPutObject(ctx, putInput, func(po *s3.PresignOptions) {
			po.Expires = expiration
		})
		if err != nil {
			return "", fmt.Errorf("failed to generate PUT presigned URL: %w", err)
		}
		
		return presignResult.URL, nil
	}
	
	// Generate GET presigned URL
	getInput := &s3.GetObjectInput{
		Bucket: aws.String(s3s.config.S3.Bucket),
		Key:    aws.String(s3Key),
	}
	
	presignResult, err := presignClient.PresignGetObject(ctx, getInput, func(po *s3.PresignOptions) {
		po.Expires = expiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate GET presigned URL: %w", err)
	}
	
	return presignResult.URL, nil
}

// List files with filtering
func (s3s *S3Storage) List(ctx context.Context, options ListOptions) ([]*FileInfo, error) {
	// For S3 storage, we rely on database metadata
	store := s3s.db.CreateStore("files")
	query := database.NewQueryBuilder()
	
	// Add storage type filter
	query.Where("storageType", "$eq", s3s.config.Type)
	
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

func (s3s *S3Storage) storeMetadata(ctx context.Context, fileInfo *FileInfo) error {
	store := s3s.db.CreateStore("files")
	
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

func (s3s *S3Storage) getMetadata(ctx context.Context, fileID string) (*FileInfo, error) {
	store := s3s.db.CreateStore("files")
	
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

func (s3s *S3Storage) deleteMetadata(ctx context.Context, fileID string) error {
	store := s3s.db.CreateStore("files")
	query := database.NewQueryBuilder().Where("id", "$eq", fileID)
	_, err := store.Remove(ctx, query)
	return err
}

func (s3s *S3Storage) generateURL(s3Key string) string {
	if s3s.config.S3.Endpoint != "" {
		// Custom endpoint (MinIO)
		endpoint := strings.TrimRight(s3s.config.S3.Endpoint, "/")
		return fmt.Sprintf("%s/%s/%s", endpoint, s3s.config.S3.Bucket, s3Key)
	}
	
	// AWS S3 URL
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", 
		s3s.config.S3.Bucket, 
		s3s.config.S3.Region, 
		s3Key,
	)
}