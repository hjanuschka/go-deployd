package resources

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	
	appcontext "github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"github.com/hjanuschka/go-deployd/internal/storage"
)

// FilesResource handles file upload and management
type FilesResource struct {
	*BaseResource
	storageManager *storage.Manager
	db             database.DatabaseInterface
}

// NewFilesResource creates a new files resource
func NewFilesResource(name string, storageManager *storage.Manager, db database.DatabaseInterface) *FilesResource {
	return &FilesResource{
		BaseResource:   NewBaseResource(name),
		storageManager: storageManager,
		db:             db,
	}
}

// ServeHTTP handles file operations
func (fr *FilesResource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Create context with authentication data
	authData := &appcontext.AuthData{
		IsAuthenticated: false,
		IsRoot:         false,
	}
	
	ctx := appcontext.New(r, w, fr, authData, true)
	ctx.Method = r.Method
	
	// Parse URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		fr.writeError(ctx, http.StatusBadRequest, "Invalid path")
		return
	}
	
	// Remove the resource name from path
	pathParts = pathParts[1:] // Remove resource name
	
	switch r.Method {
	case "GET":
		fr.handleGet(ctx, pathParts)
	case "POST":
		fr.handlePost(ctx, pathParts)
	case "DELETE":
		fr.handleDelete(ctx, pathParts)
	default:
		fr.writeError(ctx, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGet serves files or file metadata
func (fr *FilesResource) handleGet(ctx *appcontext.Context, pathParts []string) {
	if len(pathParts) == 0 {
		// List files
		fr.handleList(ctx)
		return
	}
	
	if len(pathParts) == 1 {
		fileID := pathParts[0]
		
		// Check for special operations
		if strings.Contains(ctx.Request.URL.RawQuery, "signed") {
			fr.handleSignedURL(ctx, fileID, "get")
			return
		}
		
		if strings.Contains(ctx.Request.URL.RawQuery, "info") {
			fr.handleInfo(ctx, fileID)
			return
		}
		
		// Download file
		fr.handleDownload(ctx, fileID)
		return
	}
	
	fr.writeError(ctx, http.StatusBadRequest, "Invalid path")
}

// handlePost handles file uploads and signed URL generation
func (fr *FilesResource) handlePost(ctx *appcontext.Context, pathParts []string) {
	if len(pathParts) == 0 {
		// Direct file upload
		fr.handleUpload(ctx)
		return
	}
	
	if len(pathParts) == 1 && pathParts[0] == "signed" {
		// Generate signed URL for upload
		fr.handleSignedURLGeneration(ctx)
		return
	}
	
	if len(pathParts) == 2 && pathParts[0] == "upload" {
		// Handle signed URL upload
		fileID := pathParts[1]
		fr.handleSignedUpload(ctx, fileID)
		return
	}
	
	fr.writeError(ctx, http.StatusBadRequest, "Invalid path")
}

// handleDelete removes files
func (fr *FilesResource) handleDelete(ctx *appcontext.Context, pathParts []string) {
	if len(pathParts) != 1 {
		fr.writeError(ctx, http.StatusBadRequest, "File ID required")
		return
	}
	
	fileID := pathParts[0]
	
	// Check permissions
	if !fr.canDeleteFile(ctx, fileID) {
		fr.writeError(ctx, http.StatusForbidden, "Permission denied")
		return
	}
	
	// Delete file
	if err := fr.storageManager.Delete(ctx.Request.Context(), fileID); err != nil {
		if _, ok := err.(storage.FileNotFoundError); ok {
			fr.writeError(ctx, http.StatusNotFound, "File not found")
			return
		}
		
		logging.Error("Failed to delete file", "files", map[string]interface{}{
			"fileID": fileID,
			"error":  err.Error(),
		})
		fr.writeError(ctx, http.StatusInternalServerError, "Failed to delete file")
		return
	}
	
	ctx.Response.WriteHeader(http.StatusNoContent)
}

// handleUpload processes direct file uploads
func (fr *FilesResource) handleUpload(ctx *appcontext.Context) {
	// Check content type
	contentType := ctx.Request.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		fr.writeError(ctx, http.StatusBadRequest, "Content-Type must be multipart/form-data")
		return
	}
	
	// Parse multipart form
	err := ctx.Request.ParseMultipartForm(fr.storageManager.GetConfig().MaxFileSize)
	if err != nil {
		fr.writeError(ctx, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}
	
	// Get file from form
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		fr.writeError(ctx, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()
	
	// Validate file
	if err := fr.storageManager.ValidateFile(header.Filename, header.Size); err != nil {
		if _, ok := err.(storage.FileTooLargeError); ok {
			fr.writeError(ctx, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		if _, ok := err.(storage.InvalidFileTypeError); ok {
			fr.writeError(ctx, http.StatusUnsupportedMediaType, err.Error())
			return
		}
		fr.writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	
	// Detect content type
	detectedContentType := detectContentType(header.Filename, file)
	
	// Get user ID from context
	userID := ""
	if ctx.IsAuthenticated {
		userID = ctx.UserID
	}
	
	// Upload options
	options := &storage.UploadOptions{
		ContentType: detectedContentType,
		UserID:      userID,
		Metadata:    make(map[string]interface{}),
	}
	
	// Add form metadata
	for key, values := range ctx.Request.Form {
		if key != "file" && len(values) > 0 {
			options.Metadata[key] = values[0]
		}
	}
	
	// Upload file
	fileInfo, err := fr.storageManager.Upload(ctx.Request.Context(), header.Filename, file, options)
	if err != nil {
		logging.Error("Failed to upload file", "files", map[string]interface{}{
			"filename": header.Filename,
			"error":    err.Error(),
		})
		fr.writeError(ctx, http.StatusInternalServerError, "Failed to upload file")
		return
	}
	
	// Return file info
	ctx.Response.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response).Encode(fileInfo)
}

// handleDownload serves file content
func (fr *FilesResource) handleDownload(ctx *appcontext.Context, fileID string) {
	// Check permissions
	if !fr.canAccessFile(ctx, fileID) {
		fr.writeError(ctx, http.StatusForbidden, "Permission denied")
		return
	}
	
	// Download file
	reader, fileInfo, err := fr.storageManager.Download(ctx.Request.Context(), fileID)
	if err != nil {
		if _, ok := err.(storage.FileNotFoundError); ok {
			fr.writeError(ctx, http.StatusNotFound, "File not found")
			return
		}
		
		logging.Error("Failed to download file", "files", map[string]interface{}{
			"fileID": fileID,
			"error":  err.Error(),
		})
		fr.writeError(ctx, http.StatusInternalServerError, "Failed to download file")
		return
	}
	defer reader.Close()
	
	// Set headers
	ctx.Response.Header().Set("Content-Type", fileInfo.ContentType)
	ctx.Response.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size, 10))
	ctx.Response.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileInfo.OriginalName))
	
	// Stream file
	_, err = io.Copy(ctx.Response, reader)
	if err != nil {
		logging.Error("Failed to stream file", "files", map[string]interface{}{
			"fileID": fileID,
			"error":  err.Error(),
		})
	}
}

// handleInfo returns file metadata
func (fr *FilesResource) handleInfo(ctx *appcontext.Context, fileID string) {
	// Check permissions
	if !fr.canAccessFile(ctx, fileID) {
		fr.writeError(ctx, http.StatusForbidden, "Permission denied")
		return
	}
	
	// Get file info
	fileInfo, err := fr.storageManager.GetInfo(ctx.Request.Context(), fileID)
	if err != nil {
		if _, ok := err.(storage.FileNotFoundError); ok {
			fr.writeError(ctx, http.StatusNotFound, "File not found")
			return
		}
		
		fr.writeError(ctx, http.StatusInternalServerError, "Failed to get file info")
		return
	}
	
	// Return file info
	ctx.Response.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response).Encode(fileInfo)
}

// handleList returns a list of files
func (fr *FilesResource) handleList(ctx *appcontext.Context) {
	// Parse query parameters
	query := ctx.Request.URL.Query()
	
	options := storage.ListOptions{
		Limit:  50, // Default limit
		Offset: 0,
	}
	
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			options.Limit = limit
		}
	}
	
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			options.Offset = offset
		}
	}
	
	// Get user ID for filtering
	if ctx.IsAuthenticated && !ctx.IsRoot {
		options.UserID = ctx.UserID // Non-root users can only see their own files
	} else if !ctx.IsRoot {
		// Anonymous users can't list files
		fr.writeError(ctx, http.StatusForbidden, "Authentication required")
		return
	}
	
	// List files
	files, err := fr.storageManager.List(ctx.Request.Context(), options)
	if err != nil {
		logging.Error("Failed to list files", "files", map[string]interface{}{
			"error": err.Error(),
		})
		fr.writeError(ctx, http.StatusInternalServerError, "Failed to list files")
		return
	}
	
	// Return files
	ctx.Response.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response).Encode(map[string]interface{}{
		"files":  files,
		"count":  len(files),
		"limit":  options.Limit,
		"offset": options.Offset,
	})
}

// handleSignedURL returns a signed URL for file access
func (fr *FilesResource) handleSignedURL(ctx *appcontext.Context, fileID, operation string) {
	// Check permissions
	if operation == "get" && !fr.canAccessFile(ctx, fileID) {
		fr.writeError(ctx, http.StatusForbidden, "Permission denied")
		return
	}
	
	// Parse expiration from query
	expiration := time.Duration(fr.storageManager.GetConfig().SignedURLExpiration) * time.Second
	if expStr := ctx.Request.URL.Query().Get("expires"); expStr != "" {
		if exp, err := strconv.Atoi(expStr); err == nil && exp > 0 && exp <= 86400 {
			expiration = time.Duration(exp) * time.Second
		}
	}
	
	// Generate signed URL
	options := &storage.SignedURLOptions{
		Operation:  operation,
		Expiration: expiration,
	}
	
	signedURL, err := fr.storageManager.GenerateSignedURL(ctx.Request.Context(), fileID, options)
	if err != nil {
		if _, ok := err.(storage.FileNotFoundError); ok {
			fr.writeError(ctx, http.StatusNotFound, "File not found")
			return
		}
		
		logging.Error("Failed to generate signed URL", "files", map[string]interface{}{
			"fileID": fileID,
			"error":  err.Error(),
		})
		fr.writeError(ctx, http.StatusInternalServerError, "Failed to generate signed URL")
		return
	}
	
	// Return signed URL
	ctx.Response.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response).Encode(map[string]interface{}{
		"signedUrl":  signedURL,
		"operation":  operation,
		"expiresIn":  expiration.Seconds(),
		"fileID":     fileID,
	})
}

// handleSignedURLGeneration generates signed URLs for uploads
func (fr *FilesResource) handleSignedURLGeneration(ctx *appcontext.Context) {
	// Parse request body
	var req struct {
		Filename    string `json:"filename"`
		ContentType string `json:"contentType"`
		ExpiresIn   int    `json:"expiresIn,omitempty"`
	}
	
	if err := json.NewDecoder(ctx.Request.Body).Decode(&req); err != nil {
		fr.writeError(ctx, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	
	if req.Filename == "" {
		fr.writeError(ctx, http.StatusBadRequest, "Filename is required")
		return
	}
	
	// Generate file ID
	fileID := generateFileID()
	
	// Validate file
	if err := fr.storageManager.ValidateFile(req.Filename, 0); err != nil {
		if _, ok := err.(storage.InvalidFileTypeError); ok {
			fr.writeError(ctx, http.StatusUnsupportedMediaType, err.Error())
			return
		}
		fr.writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	
	// Set expiration
	expiration := time.Duration(fr.storageManager.GetConfig().SignedURLExpiration) * time.Second
	if req.ExpiresIn > 0 && req.ExpiresIn <= 86400 {
		expiration = time.Duration(req.ExpiresIn) * time.Second
	}
	
	// Generate signed URL for upload
	options := &storage.SignedURLOptions{
		Operation:   "put",
		ContentType: req.ContentType,
		Expiration:  expiration,
	}
	
	signedURL, err := fr.storageManager.GenerateSignedURL(ctx.Request.Context(), fileID, options)
	if err != nil {
		logging.Error("Failed to generate upload signed URL", "files", map[string]interface{}{
			"filename": req.Filename,
			"error":    err.Error(),
		})
		fr.writeError(ctx, http.StatusInternalServerError, "Failed to generate signed URL")
		return
	}
	
	// Return signed URL and file info
	ctx.Response.Header().Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response).Encode(map[string]interface{}{
		"signedUrl":    signedURL,
		"fileID":       fileID,
		"filename":     req.Filename,
		"contentType":  req.ContentType,
		"expiresIn":    expiration.Seconds(),
		"completeUrl":  fmt.Sprintf("%s/%s/complete/%s", ctx.Request.URL.Scheme+"://"+ctx.Request.Host+"/"+fr.name, fr.name, fileID),
	})
}

// handleSignedUpload handles uploads via signed URLs (local storage only)
func (fr *FilesResource) handleSignedUpload(ctx *appcontext.Context, fileID string) {
	// This is mainly for local storage compatibility
	// For S3/MinIO, uploads go directly to the storage service
	
	if fr.storageManager.GetConfig().Type != "local" {
		fr.writeError(ctx, http.StatusBadRequest, "Direct upload not supported for this storage type")
		return
	}
	
	// Handle the upload similar to regular upload
	fr.handleUpload(ctx)
}

// Permission checking helpers

func (fr *FilesResource) canAccessFile(ctx *appcontext.Context, fileID string) bool {
	// Root users can access any file
	if ctx.IsRoot {
		return true
	}
	
	// Get file info to check ownership
	fileInfo, err := fr.storageManager.GetInfo(ctx.Request.Context(), fileID)
	if err != nil {
		return false
	}
	
	// If not authenticated, deny access
	if !ctx.IsAuthenticated {
		return false
	}
	
	// Check if user owns the file
	return fileInfo.UploadedBy == ctx.UserID
}

func (fr *FilesResource) canDeleteFile(ctx *appcontext.Context, fileID string) bool {
	// Same logic as access for now
	return fr.canAccessFile(ctx, fileID)
}

// Utility functions

func (fr *FilesResource) writeError(ctx *appcontext.Context, status int, message string) {
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(status)
	json.NewEncoder(ctx.Response).Encode(map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
		"status":  status,
	})
}

func detectContentType(filename string, file multipart.File) string {
	// Try to detect from filename extension
	ext := filepath.Ext(filename)
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	
	// Try to detect from file content
	buffer := make([]byte, 512)
	if n, err := file.Read(buffer); err == nil {
		contentType := http.DetectContentType(buffer[:n])
		// Reset file reader
		file.Seek(0, 0)
		return contentType
	}
	
	// Default
	return "application/octet-stream"
}

// Resource interface implementation

func (fr *FilesResource) GetName() string {
	return fr.name
}

func (fr *FilesResource) GetType() string {
	return "files"
}

func generateFileID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}