package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hjanuschka/go-deployd/internal/auth"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/server"
	"github.com/hjanuschka/go-deployd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type TestServer struct {
	Server   *server.Server
	DB       database.DatabaseInterface
	Router   *router.Router
	EventMgr *events.Manager
}

func setupTestServer(t *testing.T) *TestServer {
	// Create test database
	db := testutil.CreateTestDB(t)
	
	// Create event manager
	eventMgr := events.NewManager()
	
	// Create router
	r := router.NewRouter()
	r.SetDatabase(db)
	r.SetEventManager(eventMgr)
	r.SetJWTSecret("test-secret-key")
	
	// Create server
	srv := server.New(r, db, eventMgr)
	
	// Initialize built-in collections
	err := initializeBuiltInCollections(db)
	require.NoError(t, err)
	
	return &TestServer{
		Server:   srv,
		DB:       db,
		Router:   r,
		EventMgr: eventMgr,
	}
}

func initializeBuiltInCollections(db database.DatabaseInterface) error {
	ctx := context.Background()
	
	// Create users collection
	userStore := db.GetStore("users")
	if err := userStore.CreateTable(ctx); err != nil {
		return err
	}
	
	// Create sessions collection
	sessionStore := db.GetStore("sessions")
	if err := sessionStore.CreateTable(ctx); err != nil {
		return err
	}
	
	return nil
}

func createTestCollectionWithHooks(t *testing.T, ts *TestServer, collectionName string) {
	// Create collection
	collection := &resources.Collection{
		Name: collectionName,
		Properties: map[string]resources.Property{
			"title": {
				Name:     "title",
				Type:     "string",
				Required: true,
			},
			"status": {
				Name:     "status",
				Type:     "string",
				Required: false,
			},
			"owner": {
				Name:     "owner",
				Type:     "string",
				Required: true,
			},
			"metadata": {
				Name:     "metadata",
				Type:     "object",
				Required: false,
			},
		},
	}
	
	// Create table
	store := ts.DB.GetStore(collectionName)
	ctx := context.Background()
	err := store.CreateTable(ctx)
	require.NoError(t, err)
	
	// Create Go hooks
	tmpDir := t.TempDir()
	
	// Validate hook
	validateHook := fmt.Sprintf(`
package main

import (
	"encoding/json"
	"fmt"
)

type EventData struct {
	Data map[string]interface{} %sjson:"data"%s
}

func Validate(input string) (string, error) {
	var event EventData
	if err := json.Unmarshal([]byte(input), &event); err != nil {
		return "", err
	}
	
	if title, ok := event.Data["title"].(string); !ok || title == "" {
		return "", fmt.Errorf("title is required and cannot be empty")
	}
	
	return input, nil
}
`, "`", "`")
	
	validatePath := filepath.Join(tmpDir, "validate.go")
	err = os.WriteFile(validatePath, []byte(validateHook), 0644)
	require.NoError(t, err)
	
	// Post hook (modifies data)
	postHook := fmt.Sprintf(`
package main

import (
	"encoding/json"
	"time"
)

type EventData struct {
	Data     map[string]interface{} %sjson:"data"%s
	Modified bool                   %sjson:"modified"%s
}

func Post(input string) (string, error) {
	var event EventData
	if err := json.Unmarshal([]byte(input), &event); err != nil {
		return "", err
	}
	
	// Add metadata
	if event.Data["metadata"] == nil {
		event.Data["metadata"] = make(map[string]interface{})
	}
	
	metadata := event.Data["metadata"].(map[string]interface{})
	metadata["createdAt"] = time.Now().Format(time.RFC3339)
	metadata["version"] = "1.0"
	
	event.Modified = true
	
	output, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	
	return string(output), nil
}
`, "`", "`", "`", "`")
	
	postPath := filepath.Join(tmpDir, "post.go")
	err = os.WriteFile(postPath, []byte(postHook), 0644)
	require.NoError(t, err)
	
	// Load hooks
	err = ts.EventMgr.LoadGoScript(validatePath, collectionName, "validate")
	require.NoError(t, err)
	
	err = ts.EventMgr.LoadGoScript(postPath, collectionName, "post")
	require.NoError(t, err)
	
	// Register collection with router
	ts.Router.RegisterCollection(collection)
}

func makeRequest(t *testing.T, ts *TestServer, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}
	
	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, req)
	return w
}

func TestCompleteIntegration(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.DB.Close()
	
	collectionName := testutil.GenerateRandomName("items")
	createTestCollectionWithHooks(t, ts, collectionName)
	defer testutil.CleanupCollection(t, ts.DB, collectionName)
	
	var user1Token, user2Token string
	var user1ID, user2ID string
	
	t.Run("User Registration and Login", func(t *testing.T) {
		// Register user 1
		regData := map[string]interface{}{
			"username": testutil.GenerateRandomName("user1"),
			"email":    fmt.Sprintf("%s@test.com", testutil.GenerateRandomName("user1")),
			"password": "password123",
		}
		
		// Simulate registration (normally done through API)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(regData["password"].(string)), bcrypt.DefaultCost)
		require.NoError(t, err)
		
		userStore := ts.DB.GetStore("users")
		ctx := context.Background()
		userData := map[string]interface{}{
			"username": regData["username"],
			"email":    regData["email"],
			"password": string(hashedPassword),
			"verified": true,
		}
		
		result, err := userStore.Insert(ctx, userData)
		require.NoError(t, err)
		user1ID = result.InsertedID.(string)
		
		// Generate JWT token
		user1Token, err = auth.GenerateJWT(user1ID, regData["username"].(string), "test-secret-key")
		require.NoError(t, err)
		
		// Register user 2
		regData2 := map[string]interface{}{
			"username": testutil.GenerateRandomName("user2"),
			"email":    fmt.Sprintf("%s@test.com", testutil.GenerateRandomName("user2")),
			"password": "password456",
		}
		
		hashedPassword2, err := bcrypt.GenerateFromPassword([]byte(regData2["password"].(string)), bcrypt.DefaultCost)
		require.NoError(t, err)
		
		userData2 := map[string]interface{}{
			"username": regData2["username"],
			"email":    regData2["email"],
			"password": string(hashedPassword2),
			"verified": true,
		}
		
		result2, err := userStore.Insert(ctx, userData2)
		require.NoError(t, err)
		user2ID = result2.InsertedID.(string)
		
		user2Token, err = auth.GenerateJWT(user2ID, regData2["username"].(string), "test-secret-key")
		require.NoError(t, err)
	})
	
	var itemID string
	
	t.Run("Create Item with Validation and Hooks", func(t *testing.T) {
		// Try to create without required field
		invalidData := map[string]interface{}{
			"status": "draft",
		}
		
		w := makeRequest(t, ts, "POST", fmt.Sprintf("/%s", collectionName), invalidData, user1Token)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		// Create valid item
		validData := map[string]interface{}{
			"title":  "My First Item",
			"status": "active",
		}
		
		w = makeRequest(t, ts, "POST", fmt.Sprintf("/%s", collectionName), validData, user1Token)
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var created map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &created)
		require.NoError(t, err)
		
		// Verify hooks were executed
		assert.NotNil(t, created["_id"])
		assert.Equal(t, "My First Item", created["title"])
		assert.Equal(t, user1ID, created["owner"])
		
		// Check that post hook added metadata
		metadata, ok := created["metadata"].(map[string]interface{})
		assert.True(t, ok)
		assert.NotEmpty(t, metadata["createdAt"])
		assert.Equal(t, "1.0", metadata["version"])
		
		itemID = created["_id"].(string)
	})
	
	t.Run("Read Operations with Permissions", func(t *testing.T) {
		// User 1 can read their own item
		w := makeRequest(t, ts, "GET", fmt.Sprintf("/%s/%s", collectionName, itemID), nil, user1Token)
		assert.Equal(t, http.StatusOK, w.Code)
		
		// User 2 cannot read User 1's item
		w = makeRequest(t, ts, "GET", fmt.Sprintf("/%s/%s", collectionName, itemID), nil, user2Token)
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		// List operations only show owned items
		w = makeRequest(t, ts, "GET", fmt.Sprintf("/%s", collectionName), nil, user1Token)
		assert.Equal(t, http.StatusOK, w.Code)
		
		var items []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &items)
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, user1ID, items[0]["owner"])
	})
	
	t.Run("Update Operations with Permissions", func(t *testing.T) {
		// User 1 can update their item
		updateData := map[string]interface{}{
			"title":  "Updated Item",
			"status": "completed",
		}
		
		w := makeRequest(t, ts, "PUT", fmt.Sprintf("/%s/%s", collectionName, itemID), updateData, user1Token)
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Verify update
		w = makeRequest(t, ts, "GET", fmt.Sprintf("/%s/%s", collectionName, itemID), nil, user1Token)
		assert.Equal(t, http.StatusOK, w.Code)
		
		var updated map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &updated)
		require.NoError(t, err)
		assert.Equal(t, "Updated Item", updated["title"])
		assert.Equal(t, "completed", updated["status"])
		
		// User 2 cannot update User 1's item
		w = makeRequest(t, ts, "PUT", fmt.Sprintf("/%s/%s", collectionName, itemID), updateData, user2Token)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
	
	t.Run("Delete Operations with Permissions", func(t *testing.T) {
		// User 2 cannot delete User 1's item
		w := makeRequest(t, ts, "DELETE", fmt.Sprintf("/%s/%s", collectionName, itemID), nil, user2Token)
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		// User 1 can delete their item
		w = makeRequest(t, ts, "DELETE", fmt.Sprintf("/%s/%s", collectionName, itemID), nil, user1Token)
		assert.Equal(t, http.StatusNoContent, w.Code)
		
		// Verify deletion
		w = makeRequest(t, ts, "GET", fmt.Sprintf("/%s/%s", collectionName, itemID), nil, user1Token)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	
	t.Run("Session Management", func(t *testing.T) {
		// Create session
		sessionStore := ts.DB.GetStore("sessions")
		ctx := context.Background()
		
		sessionToken := testutil.GenerateRandomName("session")
		sessionData := map[string]interface{}{
			"token":     sessionToken,
			"userId":    user1ID,
			"createdAt": time.Now(),
			"expiresAt": time.Now().Add(24 * time.Hour),
			"active":    true,
		}
		
		result, err := sessionStore.Insert(ctx, sessionData)
		require.NoError(t, err)
		assert.NotNil(t, result.InsertedID)
		
		// Verify session is active
		query := ts.DB.CreateQuery().
			Where("token", "=", sessionToken).
			Where("active", "=", true).
			Where("expiresAt", ">", time.Now())
		
		sessions, err := sessionStore.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		
		// Invalidate session
		update := ts.DB.CreateUpdate().Set("active", false)
		updateResult, err := sessionStore.Update(ctx, query, update)
		require.NoError(t, err)
		assert.Greater(t, updateResult.ModifiedCount, int64(0))
	})
}

func TestMultipleDatabaseSupport(t *testing.T) {
	// This test ensures our code works with different database backends
	dbTypes := []string{"sqlite"}
	
	// Add other databases if available
	if os.Getenv("TEST_MONGO_URL") != "" {
		dbTypes = append(dbTypes, "mongodb")
	}
	if os.Getenv("TEST_MYSQL_DSN") != "" {
		dbTypes = append(dbTypes, "mysql")
	}
	
	for _, dbType := range dbTypes {
		t.Run(fmt.Sprintf("Database_%s", dbType), func(t *testing.T) {
			// Set database type for test
			oldDBType := os.Getenv("TEST_DB")
			os.Setenv("TEST_DB", dbType)
			defer os.Setenv("TEST_DB", oldDBType)
			
			ts := setupTestServer(t)
			defer ts.DB.Close()
			
			// Create test collection
			collectionName := testutil.GenerateRandomName("db_test")
			collection := testutil.CreateTestCollection(t, ts.DB, collectionName)
			defer testutil.CleanupCollection(t, ts.DB, collectionName)
			
			store := ts.DB.GetStore(collectionName)
			ctx := context.Background()
			
			// Test CRUD operations
			doc := map[string]interface{}{
				"name":  "Test Doc",
				"owner": "testuser",
				"data": map[string]interface{}{
					"nested": "value",
				},
			}
			
			// Create
			result, err := store.Insert(ctx, doc)
			require.NoError(t, err)
			assert.NotNil(t, result.InsertedID)
			
			// Read
			query := ts.DB.CreateQuery().Where("_id", "=", result.InsertedID)
			docs, err := store.Find(ctx, query)
			require.NoError(t, err)
			assert.Len(t, docs, 1)
			assert.Equal(t, "Test Doc", docs[0]["name"])
			
			// Update
			update := ts.DB.CreateUpdate().Set("name", "Updated Doc")
			updateResult, err := store.Update(ctx, query, update)
			require.NoError(t, err)
			assert.Greater(t, updateResult.ModifiedCount, int64(0))
			
			// Delete
			deleteResult, err := store.Delete(ctx, query)
			require.NoError(t, err)
			assert.Greater(t, deleteResult.DeletedCount, int64(0))
		})
	}
}