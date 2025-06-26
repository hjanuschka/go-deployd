package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hjanuschka/go-deployd/internal/auth"
	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(t *testing.T, db database.DatabaseInterface) *router.Router {
	r := router.NewRouter()
	r.SetDatabase(db)
	
	// Setup JWT for authentication
	jwtSecret := "test-secret-key"
	r.SetJWTSecret(jwtSecret)
	
	return r
}

func createAuthToken(t *testing.T, userID, username string) string {
	token, err := auth.GenerateJWT(userID, username, "test-secret-key")
	require.NoError(t, err)
	return token
}

func makeAuthenticatedRequest(t *testing.T, method, url string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}
	
	req := httptest.NewRequest(method, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	w := httptest.NewRecorder()
	return w
}

func TestCRUDOperations(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	r := setupTestRouter(t, db)
	
	// Create test users
	user1 := testutil.CreateTestUser(t, db)
	user1.Token = createAuthToken(t, user1.ID, user1.Username)
	
	user2 := testutil.CreateTestUser(t, db)
	user2.Token = createAuthToken(t, user2.ID, user2.Username)
	
	// Create test collection
	collectionName := testutil.GenerateRandomName("items")
	collection := testutil.CreateTestCollection(t, db, collectionName)
	defer testutil.CleanupCollection(t, db, collectionName)
	
	// Register collection with router
	r.RegisterCollection(collection)

	t.Run("Create document", func(t *testing.T) {
		docData := map[string]interface{}{
			"name": "My Item",
			"data": map[string]interface{}{
				"description": "Test item",
				"price":       99.99,
			},
		}
		
		w := makeAuthenticatedRequest(t, "POST", fmt.Sprintf("/%s", collectionName), docData, user1.Token)
		r.ServeHTTP(w, nil)
		
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.NotEmpty(t, response["_id"])
		assert.Equal(t, "My Item", response["name"])
		assert.Equal(t, user1.ID, response["owner"])
	})

	t.Run("Read documents", func(t *testing.T) {
		// Create some test documents
		docs := []map[string]interface{}{
			{"name": "Item 1", "owner": user1.ID},
			{"name": "Item 2", "owner": user1.ID},
			{"name": "Item 3", "owner": user2.ID},
		}
		
		store := db.GetStore(collectionName)
		ctx := context.Background()
		
		for _, doc := range docs {
			_, err := store.Insert(ctx, doc)
			require.NoError(t, err)
		}
		
		// User1 should only see their own documents
		w := makeAuthenticatedRequest(t, "GET", fmt.Sprintf("/%s", collectionName), nil, user1.Token)
		r.ServeHTTP(w, nil)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var items []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &items)
		require.NoError(t, err)
		
		assert.Len(t, items, 2)
		for _, item := range items {
			assert.Equal(t, user1.ID, item["owner"])
		}
	})

	t.Run("Read single document", func(t *testing.T) {
		// Create a document
		doc := map[string]interface{}{
			"name":  "Single Item",
			"owner": user1.ID,
		}
		
		store := db.GetStore(collectionName)
		ctx := context.Background()
		result, err := store.Insert(ctx, doc)
		require.NoError(t, err)
		
		docID := result.InsertedID.(string)
		
		// Owner can read
		w := makeAuthenticatedRequest(t, "GET", fmt.Sprintf("/%s/%s", collectionName, docID), nil, user1.Token)
		r.ServeHTTP(w, nil)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var item map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &item)
		require.NoError(t, err)
		assert.Equal(t, "Single Item", item["name"])
		
		// Non-owner cannot read
		w2 := makeAuthenticatedRequest(t, "GET", fmt.Sprintf("/%s/%s", collectionName, docID), nil, user2.Token)
		r.ServeHTTP(w2, nil)
		
		assert.Equal(t, http.StatusForbidden, w2.Code)
	})

	t.Run("Update document", func(t *testing.T) {
		// Create a document
		doc := map[string]interface{}{
			"name":  "Original Name",
			"owner": user1.ID,
		}
		
		store := db.GetStore(collectionName)
		ctx := context.Background()
		result, err := store.Insert(ctx, doc)
		require.NoError(t, err)
		
		docID := result.InsertedID.(string)
		
		// Owner can update
		updateData := map[string]interface{}{
			"name": "Updated Name",
			"data": map[string]interface{}{
				"updated": true,
			},
		}
		
		w := makeAuthenticatedRequest(t, "PUT", fmt.Sprintf("/%s/%s", collectionName, docID), updateData, user1.Token)
		r.ServeHTTP(w, nil)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Verify update
		query := db.CreateQuery().Where("_id", "=", docID)
		docs, err := store.Find(ctx, query)
		require.NoError(t, err)
		require.Len(t, docs, 1)
		
		assert.Equal(t, "Updated Name", docs[0]["name"])
		assert.Equal(t, user1.ID, docs[0]["owner"]) // Owner should not change
		
		// Non-owner cannot update
		w2 := makeAuthenticatedRequest(t, "PUT", fmt.Sprintf("/%s/%s", collectionName, docID), updateData, user2.Token)
		r.ServeHTTP(w2, nil)
		
		assert.Equal(t, http.StatusForbidden, w2.Code)
	})

	t.Run("Delete document", func(t *testing.T) {
		// Create a document
		doc := map[string]interface{}{
			"name":  "To Delete",
			"owner": user1.ID,
		}
		
		store := db.GetStore(collectionName)
		ctx := context.Background()
		result, err := store.Insert(ctx, doc)
		require.NoError(t, err)
		
		docID := result.InsertedID.(string)
		
		// Non-owner cannot delete
		w := makeAuthenticatedRequest(t, "DELETE", fmt.Sprintf("/%s/%s", collectionName, docID), nil, user2.Token)
		r.ServeHTTP(w, nil)
		
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		// Owner can delete
		w2 := makeAuthenticatedRequest(t, "DELETE", fmt.Sprintf("/%s/%s", collectionName, docID), nil, user1.Token)
		r.ServeHTTP(w2, nil)
		
		assert.Equal(t, http.StatusNoContent, w2.Code)
		
		// Verify deletion
		query := db.CreateQuery().Where("_id", "=", docID)
		docs, err := store.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, docs, 0)
	})

	t.Run("Unauthenticated requests", func(t *testing.T) {
		// All CRUD operations should require authentication
		
		w := makeAuthenticatedRequest(t, "GET", fmt.Sprintf("/%s", collectionName), nil, "")
		r.ServeHTTP(w, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		w = makeAuthenticatedRequest(t, "POST", fmt.Sprintf("/%s", collectionName), map[string]interface{}{"name": "test"}, "")
		r.ServeHTTP(w, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		w = makeAuthenticatedRequest(t, "PUT", fmt.Sprintf("/%s/123", collectionName), map[string]interface{}{"name": "test"}, "")
		r.ServeHTTP(w, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		w = makeAuthenticatedRequest(t, "DELETE", fmt.Sprintf("/%s/123", collectionName), nil, "")
		r.ServeHTTP(w, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestUserPermissions(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	r := setupTestRouter(t, db)
	
	// Create test users
	user1 := testutil.CreateTestUser(t, db)
	user1.Token = createAuthToken(t, user1.ID, user1.Username)
	
	user2 := testutil.CreateTestUser(t, db)
	user2.Token = createAuthToken(t, user2.ID, user2.Username)
	
	adminUser := testutil.CreateTestUser(t, db)
	adminUser.Token = createAuthToken(t, adminUser.ID, adminUser.Username)
	// Mark as admin in database
	userStore := db.GetStore("users")
	ctx := context.Background()
	update := db.CreateUpdate().Set("isAdmin", true)
	query := db.CreateQuery().Where("_id", "=", adminUser.ID)
	_, err := userStore.Update(ctx, query, update)
	require.NoError(t, err)
	
	// Create test collection
	collectionName := testutil.GenerateRandomName("protected_items")
	collection := testutil.CreateTestCollection(t, db, collectionName)
	defer testutil.CleanupCollection(t, db, collectionName)
	
	r.RegisterCollection(collection)

	t.Run("Users can only access own documents", func(t *testing.T) {
		// Create documents for different users
		store := db.GetStore(collectionName)
		
		doc1 := map[string]interface{}{
			"name":  "User1 Doc",
			"owner": user1.ID,
		}
		result1, err := store.Insert(ctx, doc1)
		require.NoError(t, err)
		
		doc2 := map[string]interface{}{
			"name":  "User2 Doc",
			"owner": user2.ID,
		}
		result2, err := store.Insert(ctx, doc2)
		require.NoError(t, err)
		
		// User1 can only see their document
		w := makeAuthenticatedRequest(t, "GET", fmt.Sprintf("/%s", collectionName), nil, user1.Token)
		r.ServeHTTP(w, nil)
		
		var items []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &items)
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "User1 Doc", items[0]["name"])
		
		// User1 cannot access User2's document
		w = makeAuthenticatedRequest(t, "GET", fmt.Sprintf("/%s/%s", collectionName, result2.InsertedID), nil, user1.Token)
		r.ServeHTTP(w, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Admin can access all documents", func(t *testing.T) {
		// Admin should see all documents
		w := makeAuthenticatedRequest(t, "GET", fmt.Sprintf("/%s", collectionName), nil, adminUser.Token)
		r.ServeHTTP(w, nil)
		
		var items []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &items)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 2)
	})

	t.Run("Owner field cannot be modified", func(t *testing.T) {
		// Create a document
		store := db.GetStore(collectionName)
		doc := map[string]interface{}{
			"name":  "Owner Test",
			"owner": user1.ID,
		}
		result, err := store.Insert(ctx, doc)
		require.NoError(t, err)
		
		docID := result.InsertedID.(string)
		
		// Try to change owner
		updateData := map[string]interface{}{
			"name":  "Updated",
			"owner": user2.ID, // Try to change owner
		}
		
		w := makeAuthenticatedRequest(t, "PUT", fmt.Sprintf("/%s/%s", collectionName, docID), updateData, user1.Token)
		r.ServeHTTP(w, nil)
		
		// Verify owner didn't change
		query := db.CreateQuery().Where("_id", "=", docID)
		docs, err := store.Find(ctx, query)
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, user1.ID, docs[0]["owner"])
	})
}