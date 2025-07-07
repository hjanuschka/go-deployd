package context

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Context struct {
	Request     *http.Request
	Response    http.ResponseWriter
	Resource    Resource
	Router      Router
	URL         string
	Query       map[string]interface{}
	Body        map[string]interface{}
	Method      string
	Development bool
	// JWT Authentication data
	UserID          string
	Username        string
	IsRoot          bool
	IsAuthenticated bool
	ctx             context.Context
	// Internal API support
	HTTPHandler http.Handler
}

type Resource interface {
	GetName() string
	GetPath() string
}

type Router interface {
	Route(ctx *Context) error
}

// AuthData contains authentication information
type AuthData struct {
	UserID          string
	Username        string
	IsRoot          bool
	IsAuthenticated bool
}

func New(req *http.Request, res http.ResponseWriter, resource Resource, auth *AuthData, development bool) *Context {
	ctx := &Context{
		Request:     req,
		Response:    res,
		Resource:    resource,
		Method:      req.Method,
		Development: development,
		ctx:         req.Context(),
	}

	// Set authentication data
	if auth != nil {
		ctx.UserID = auth.UserID
		ctx.Username = auth.Username
		ctx.IsRoot = auth.IsRoot
		ctx.IsAuthenticated = auth.IsAuthenticated
	}

	ctx.parseURL()
	ctx.parseQuery()
	ctx.parseBody()

	return ctx
}

func (c *Context) parseURL() {
	c.URL = c.Request.URL.Path
	if c.Resource != nil {
		resourcePath := c.Resource.GetPath()
		if strings.HasPrefix(c.URL, resourcePath) {
			c.URL = strings.TrimPrefix(c.URL, resourcePath)
		}
	}
	if c.URL == "" {
		c.URL = "/"
	}
}

func (c *Context) parseQuery() {
	c.Query = make(map[string]interface{})

	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			// Try to parse as different types
			value := values[0]

			// Try to parse as JSON
			if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
				var jsonValue interface{}
				if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
					c.Query[key] = jsonValue
					continue
				}
			}

			// Try to parse as number
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				c.Query[key] = num
				continue
			}

			// Try to parse as boolean
			if value == "true" {
				c.Query[key] = true
				continue
			}
			if value == "false" {
				c.Query[key] = false
				continue
			}

			// Default to string
			c.Query[key] = value
		} else {
			// Multiple values as array
			var convertedValues []interface{}
			for _, v := range values {
				convertedValues = append(convertedValues, v)
			}
			c.Query[key] = convertedValues
		}
	}
}

func (c *Context) parseBody() {
	c.Body = make(map[string]interface{})

	if c.Request.Body == nil {
		return
	}

	contentType := c.Request.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var jsonBody map[string]interface{}
		err := json.NewDecoder(c.Request.Body).Decode(&jsonBody)
		if err == nil {
			for k, v := range jsonBody {
				c.Body[k] = v
			}
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := c.Request.ParseForm(); err == nil {
			for key, values := range c.Request.PostForm {
				if len(values) == 1 {
					c.Body[key] = values[0]
				} else {
					c.Body[key] = values
				}
			}
		}
	} else if strings.Contains(contentType, "multipart/form-data") {
		c.parseMultipartForm()
	}
}

func (c *Context) parseMultipartForm() {
	// Parse multipart form with 32MB max memory
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		return
	}

	// Handle form fields
	if c.Request.MultipartForm.Value != nil {
		for key, values := range c.Request.MultipartForm.Value {
			if len(values) == 1 {
				c.Body[key] = values[0]
			} else {
				c.Body[key] = values
			}
		}
	}

	// Handle file uploads
	if c.Request.MultipartForm.File != nil {
		for fieldName, files := range c.Request.MultipartForm.File {
			if len(files) == 0 {
				continue
			}

			// For files collection, we expect the file field to be named "file"
			if fieldName == "file" && len(files) == 1 {
				if err := c.processFileUpload(files[0]); err != nil {
					// Log error but continue processing
					continue
				}
			} else {
				// Handle multiple files or other field names
				var fileInfos []map[string]interface{}
				for _, fileHeader := range files {
					if fileInfo := c.createFileInfo(fileHeader); fileInfo != nil {
						fileInfos = append(fileInfos, fileInfo)
					}
				}
				if len(fileInfos) > 0 {
					c.Body[fieldName] = fileInfos
				}
			}
		}
	}
}

func (c *Context) processFileUpload(fileHeader *multipart.FileHeader) error {
	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return err
	}

	// Generate unique filename
	hash := md5.New()
	hash.Write([]byte(fmt.Sprintf("%s_%d", fileHeader.Filename, time.Now().UnixNano())))
	hashString := fmt.Sprintf("%x", hash.Sum(nil))
	
	// Keep original extension
	ext := filepath.Ext(fileHeader.Filename)
	filename := hashString + ext
	filePath := filepath.Join(uploadsDir, filename)

	// Create the destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy file contents
	size, err := io.Copy(dst, file)
	if err != nil {
		// Clean up partial file
		os.Remove(filePath)
		return err
	}

	// Generate file URL (relative to server root)
	fileURL := "/" + filePath

	// Create file metadata for the files collection
	c.Body["id"] = hashString
	c.Body["filename"] = filename
	c.Body["originalName"] = fileHeader.Filename
	c.Body["contentType"] = fileHeader.Header.Get("Content-Type")
	c.Body["size"] = float64(size) // Use float64 for JSON compatibility
	c.Body["storageType"] = "local"
	c.Body["path"] = filePath
	c.Body["url"] = fileURL
	c.Body["uploadedAt"] = time.Now().Format(time.RFC3339)
	
	return nil
}

func (c *Context) createFileInfo(fileHeader *multipart.FileHeader) map[string]interface{} {
	if fileHeader == nil {
		return nil
	}

	// For non-files collection uploads, just return basic info
	return map[string]interface{}{
		"filename":     fileHeader.Filename,
		"contentType":  fileHeader.Header.Get("Content-Type"),
		"size":         fileHeader.Size,
	}
}

func (c *Context) ParseJSON(v interface{}) error {
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func (c *Context) WriteJSON(data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.Response).Encode(data)
}

func (c *Context) WriteError(statusCode int, message string) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(statusCode)
	return json.NewEncoder(c.Response).Encode(map[string]interface{}{
		"error":   true,
		"message": message,
		"status":  statusCode,
	})
}

func (c *Context) GetID() string {
	// Try to get ID from URL path
	if c.URL != "/" {
		parts := strings.Split(strings.Trim(c.URL, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Try to get ID from query
	if id, exists := c.Query["id"]; exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}

	// Try to get ID from body
	if id, exists := c.Body["id"]; exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}

	return ""
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) Done(err error, result interface{}) {
	if err != nil {
		c.WriteError(500, err.Error())
		return
	}

	if result != nil {
		c.WriteJSON(result)
	} else {
		c.Response.WriteHeader(204) // No Content
	}
}
