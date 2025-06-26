package context_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hjanuschka/go-deployd/internal/context"
	"github.com/stretchr/testify/assert"
)

// Mock resource for testing
type mockResource struct {
	name string
	path string
}

func (m *mockResource) GetName() string { return m.name }
func (m *mockResource) GetPath() string { return m.path }

// Mock router for testing
type mockRouter struct{}

func (m *mockRouter) Route(ctx *context.Context) error { return nil }

func TestContextCreation(t *testing.T) {
	t.Run("New context with all parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users?limit=10", nil)
		res := httptest.NewRecorder()
		resource := &mockResource{name: "users", path: "/api/users"}
		auth := &context.AuthData{
			UserID:          "user123",
			Username:        "testuser",
			IsRoot:          true,
			IsAuthenticated: true,
		}

		ctx := context.New(req, res, resource, auth, true)

		assert.Equal(t, req, ctx.Request)
		assert.Equal(t, res, ctx.Response)
		assert.Equal(t, resource, ctx.Resource)
		assert.Equal(t, "GET", ctx.Method)
		assert.True(t, ctx.Development)
		assert.Equal(t, "user123", ctx.UserID)
		assert.Equal(t, "testuser", ctx.Username)
		assert.True(t, ctx.IsRoot)
		assert.True(t, ctx.IsAuthenticated)
		assert.NotNil(t, ctx.Context())
	})

	t.Run("New context without auth", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/items", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.Equal(t, "POST", ctx.Method)
		assert.False(t, ctx.Development)
		assert.Empty(t, ctx.UserID)
		assert.Empty(t, ctx.Username)
		assert.False(t, ctx.IsRoot)
		assert.False(t, ctx.IsAuthenticated)
	})
}

func TestURLParsing(t *testing.T) {
	t.Run("Parse URL with resource path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/123", nil)
		res := httptest.NewRecorder()
		resource := &mockResource{name: "users", path: "/api/users"}

		ctx := context.New(req, res, resource, nil, false)

		assert.Equal(t, "/123", ctx.URL)
	})

	t.Run("Parse URL without resource", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/items", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.Equal(t, "/api/items", ctx.URL)
	})

	t.Run("Parse empty URL path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()
		resource := &mockResource{name: "users", path: "/api/users"}

		ctx := context.New(req, res, resource, nil, false)

		assert.Equal(t, "/", ctx.URL)
	})
}

func TestQueryParsing(t *testing.T) {
	t.Run("Parse string query parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users?name=john&status=active", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.Equal(t, "john", ctx.Query["name"])
		assert.Equal(t, "active", ctx.Query["status"])
	})

	t.Run("Parse numeric query parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users?limit=10&offset=20.5", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.Equal(t, float64(10), ctx.Query["limit"])
		assert.Equal(t, float64(20.5), ctx.Query["offset"])
	})

	t.Run("Parse boolean query parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users?active=true&deleted=false", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.Equal(t, true, ctx.Query["active"])
		assert.Equal(t, false, ctx.Query["deleted"])
	})

	t.Run("Parse JSON query parameters", func(t *testing.T) {
		jsonValue := url.QueryEscape(`{"key":"value","num":42}`)
		req := httptest.NewRequest("GET", "/api/users?filter="+jsonValue, nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		filter, ok := ctx.Query["filter"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value", filter["key"])
		assert.Equal(t, float64(42), filter["num"])
	})

	t.Run("Parse multiple values as array", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users?tags=web&tags=api&tags=backend", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		tags, ok := ctx.Query["tags"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, []interface{}{"web", "api", "backend"}, tags)
	})
}

func TestBodyParsing(t *testing.T) {
	t.Run("Parse JSON body", func(t *testing.T) {
		body := map[string]interface{}{
			"name":   "John Doe",
			"age":    30,
			"active": true,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.Equal(t, "John Doe", ctx.Body["name"])
		assert.Equal(t, float64(30), ctx.Body["age"])
		assert.Equal(t, true, ctx.Body["active"])
	})

	t.Run("Parse form-encoded body", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("username", "johndoe")
		formData.Set("email", "john@example.com")
		formData.Add("roles", "user")
		formData.Add("roles", "admin")

		req := httptest.NewRequest("POST", "/api/users", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.Equal(t, "johndoe", ctx.Body["username"])
		assert.Equal(t, "john@example.com", ctx.Body["email"])
		roles, ok := ctx.Body["roles"].([]string)
		assert.True(t, ok)
		assert.Equal(t, []string{"user", "admin"}, roles)
	})

	t.Run("Handle empty body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.NotNil(t, ctx.Body)
		assert.Equal(t, 0, len(ctx.Body))
	})

	t.Run("Handle unsupported content type", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/users", strings.NewReader("plain text"))
		req.Header.Set("Content-Type", "text/plain")
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		assert.NotNil(t, ctx.Body)
		assert.Equal(t, 0, len(ctx.Body))
	})
}

func TestContextMethods(t *testing.T) {
	t.Run("ParseJSON", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		// Create fresh request for ParseJSON since body is consumed during context creation
		req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		// Since body is already parsed during context creation, verify it was parsed correctly
		assert.Equal(t, "Test User", ctx.Body["name"])
		assert.Equal(t, "test@example.com", ctx.Body["email"])

		// Test ParseJSON with a fresh request
		freshReq := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonBody))
		ctx.Request = freshReq

		var parsed map[string]interface{}
		err := ctx.ParseJSON(&parsed)

		assert.NoError(t, err)
		assert.Equal(t, "Test User", parsed["name"])
		assert.Equal(t, "test@example.com", parsed["email"])
	})

	t.Run("WriteJSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		data := map[string]interface{}{
			"message": "success",
			"count":   42,
		}

		err := ctx.WriteJSON(data)

		assert.NoError(t, err)
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(res.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])
		assert.Equal(t, float64(42), response["count"])
	})

	t.Run("WriteError", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		err := ctx.WriteError(404, "Not found")

		assert.NoError(t, err)
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 404, res.Code)

		var response map[string]interface{}
		err = json.Unmarshal(res.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, true, response["error"])
		assert.Equal(t, "Not found", response["message"])
		assert.Equal(t, float64(404), response["status"])
	})
}

func TestGetID(t *testing.T) {
	t.Run("Get ID from URL path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()
		resource := &mockResource{name: "users", path: "/api/users"}

		ctx := context.New(req, res, resource, nil, false)
		// Manually set URL to simulate parsed URL
		ctx.URL = "/user123"

		id := ctx.GetID()
		assert.Equal(t, "user123", id)
	})

	t.Run("Get ID from query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users?id=query123", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)
		// Set URL to root to avoid extracting "users" as ID from path
		ctx.URL = "/"

		id := ctx.GetID()
		assert.Equal(t, "query123", id)
	})

	t.Run("Get ID from request body", func(t *testing.T) {
		body := map[string]interface{}{
			"id":   "body123",
			"name": "Test User",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)
		// Set URL to root to avoid extracting "users" as ID from path
		ctx.URL = "/"

		id := ctx.GetID()
		assert.Equal(t, "body123", id)
	})

	t.Run("No ID found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()
		resource := &mockResource{name: "users", path: "/api/users"}

		ctx := context.New(req, res, resource, nil, false)

		id := ctx.GetID()
		assert.Equal(t, "", id)
	})

	t.Run("ID precedence: URL > Query > Body", func(t *testing.T) {
		body := map[string]interface{}{
			"id": "body123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/users?id=query123", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		resource := &mockResource{name: "users", path: "/api/users"}

		ctx := context.New(req, res, resource, nil, false)
		// Manually set URL to simulate URL-based ID
		ctx.URL = "/url123"

		id := ctx.GetID()
		assert.Equal(t, "url123", id) // URL takes precedence
	})
}

func TestDone(t *testing.T) {
	t.Run("Done with error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		ctx.Done(assert.AnError, nil)

		assert.Equal(t, 500, res.Code)
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(res.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, true, response["error"])
		assert.Equal(t, assert.AnError.Error(), response["message"])
	})

	t.Run("Done with result", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		result := map[string]interface{}{
			"success": true,
			"data":    "test",
		}

		ctx.Done(nil, result)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(res.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, true, response["success"])
		assert.Equal(t, "test", response["data"])
	})

	t.Run("Done with no result", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/123", nil)
		res := httptest.NewRecorder()

		ctx := context.New(req, res, nil, nil, false)

		ctx.Done(nil, nil)

		assert.Equal(t, 204, res.Code) // No Content
	})
}

func TestAuthData(t *testing.T) {
	t.Run("AuthData struct", func(t *testing.T) {
		auth := &context.AuthData{
			UserID:          "test123",
			Username:        "testuser",
			IsRoot:          false,
			IsAuthenticated: true,
		}

		assert.Equal(t, "test123", auth.UserID)
		assert.Equal(t, "testuser", auth.Username)
		assert.False(t, auth.IsRoot)
		assert.True(t, auth.IsAuthenticated)
	})
}

func TestIntegration(t *testing.T) {
	t.Run("Complete request processing flow", func(t *testing.T) {
		// Create a realistic request with all components
		body := map[string]interface{}{
			"name":   "Integration Test",
			"status": "active",
			"metadata": map[string]interface{}{
				"source": "test",
				"priority": 1,
			},
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/users/123?expand=true&fields=name,email", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")
		res := httptest.NewRecorder()

		resource := &mockResource{name: "users", path: "/api/v1/users"}
		auth := &context.AuthData{
			UserID:          "current_user",
			Username:        "admin",
			IsRoot:          true,
			IsAuthenticated: true,
		}

		ctx := context.New(req, res, resource, auth, true)

		// Verify all parsing worked correctly
		assert.Equal(t, "POST", ctx.Method)
		assert.Equal(t, "/123", ctx.URL)
		assert.Equal(t, true, ctx.Query["expand"])
		assert.Equal(t, "name,email", ctx.Query["fields"])
		assert.Equal(t, "Integration Test", ctx.Body["name"])
		assert.Equal(t, "active", ctx.Body["status"])
		assert.Equal(t, "current_user", ctx.UserID)
		assert.True(t, ctx.IsRoot)

		// Test ID extraction
		id := ctx.GetID()
		assert.Equal(t, "123", id)

		// Test response writing
		response := map[string]interface{}{
			"id":     "123",
			"name":   "Integration Test",
			"status": "processed",
		}
		err := ctx.WriteJSON(response)
		assert.NoError(t, err)

		var written map[string]interface{}
		err = json.Unmarshal(res.Body.Bytes(), &written)
		assert.NoError(t, err)
		assert.Equal(t, "123", written["id"])
		assert.Equal(t, "processed", written["status"])
	})
}