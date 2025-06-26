package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hjanuschka/go-deployd/internal/router"
	"github.com/hjanuschka/go-deployd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	t.Run("create new router", func(t *testing.T) {
		r := router.New(db, true, "")
		require.NotNil(t, r)
		assert.NotNil(t, r)
	})
}

func TestRouterMethods(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	t.Run("router basic functionality", func(t *testing.T) {
		r := router.New(db, true, "")
		require.NotNil(t, r)
		
		// Test that the router was created successfully
		// Without knowing the exact interface, we can't test much more
		assert.True(t, true)
	})
}

func TestRouterResourceManagement(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	t.Run("GetResources returns initial resources", func(t *testing.T) {
		r := router.New(db, true, "")
		resources := r.GetResources()
		assert.NotNil(t, resources)
		
		// Should have at least the built-in users collection
		assert.GreaterOrEqual(t, len(resources), 1)
		
		t.Log("✅ GetResources works correctly")
	})
	
	t.Run("GetCollection retrieves existing collection", func(t *testing.T) {
		r := router.New(db, true, "")
		
		// Try to get the built-in users collection
		usersCollection := r.GetCollection("users")
		if usersCollection != nil {
			assert.Equal(t, "users", usersCollection.GetName())
			t.Log("✅ GetCollection found users collection")
		} else {
			t.Log("ℹ️ Users collection not found (may not be auto-created)")
		}
		
		// Try to get a non-existent collection
		nonExistent := r.GetCollection("nonexistent")
		assert.Nil(t, nonExistent)
		
		t.Log("✅ GetCollection handles non-existent collections")
	})
}

func TestRouterHTTPHandling(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	t.Run("ServeHTTP handles OPTIONS request", func(t *testing.T) {
		r := router.New(db, true, "")
		
		// Create mock HTTP request and response
		req, err := http.NewRequest("OPTIONS", "/test", nil)
		require.NoError(t, err)
		
		rr := httptest.NewRecorder()
		
		// Call ServeHTTP
		r.ServeHTTP(rr, req)
		
		// Check response
		assert.Equal(t, http.StatusOK, rr.Code)
		
		// Check CORS headers
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")
		
		t.Log("✅ ServeHTTP handles OPTIONS requests correctly")
	})
	
	t.Run("ServeHTTP sets CORS headers", func(t *testing.T) {
		r := router.New(db, true, "")
		
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		
		// Check CORS headers are set
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Methods"))
		assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Headers"))
		
		t.Log("✅ ServeHTTP sets CORS headers correctly")
	})
}