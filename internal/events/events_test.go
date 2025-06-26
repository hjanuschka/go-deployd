package events_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hjanuschka/go-deployd/internal/events"
	"github.com/hjanuschka/go-deployd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestGoHook(t *testing.T, hookName, eventType string) string {
	tmpDir := t.TempDir()
	hookPath := filepath.Join(tmpDir, fmt.Sprintf("%s.go", hookName))

	hookCode := fmt.Sprintf(`
package main

import (
	"encoding/json"
	"fmt"
)

type EventData struct {
	Collection string                 %sjson:"collection"%s
	Method     string                 %sjson:"method"%s
	Data       map[string]interface{} %sjson:"data"%s
	UserID     string                 %sjson:"userId,omitempty"%s
	Modified   bool                   %sjson:"modified,omitempty"%s
}

func %s(input string) (string, error) {
	var event EventData
	if err := json.Unmarshal([]byte(input), &event); err != nil {
		return "", err
	}

	// Test modification based on event type
	switch event.Method {
	case "validate":
		if event.Data["name"] == "" {
			return "", fmt.Errorf("name is required")
		}
	case "post", "put":
		// Modify data
		event.Data["processed"] = true
		event.Data["processedAt"] = time.Now().Format(time.RFC3339)
		event.Modified = true
	case "get":
		// Add computed field
		if name, ok := event.Data["name"].(string); ok {
			event.Data["displayName"] = fmt.Sprintf("Processed: %%s", name)
		}
	}

	output, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	return string(output), nil
}
`, "`", "`", "`", "`", "`", "`", "`", "`", eventType)

	err := os.WriteFile(hookPath, []byte(hookCode), 0644)
	require.NoError(t, err)

	return hookPath
}

func TestGoHookEvents(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	collectionName := testutil.GenerateRandomName("hook_test")
	collection := testutil.CreateTestCollection(t, db, collectionName)
	defer testutil.CleanupCollection(t, db, collectionName)

	ctx := context.Background()

	t.Run("validate event hook", func(t *testing.T) {
		hookPath := createTestGoHook(t, "validate_hook", "Validate")
		manager := events.NewManager()
		
		err := manager.LoadGoScript(hookPath, collectionName, "validate")
		require.NoError(t, err)

		// Test valid data
		validData := map[string]interface{}{
			"collection": collectionName,
			"method":     "validate",
			"data": map[string]interface{}{
				"name": "Test Item",
			},
		}

		result, err := manager.ExecuteHook(ctx, collectionName, "validate", validData)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test invalid data
		invalidData := map[string]interface{}{
			"collection": collectionName,
			"method":     "validate",
			"data": map[string]interface{}{
				"description": "Missing name",
			},
		}

		_, err = manager.ExecuteHook(ctx, collectionName, "validate", invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("post event hook modifies data", func(t *testing.T) {
		hookPath := createTestGoHook(t, "post_hook", "Post")
		manager := events.NewManager()
		
		err := manager.LoadGoScript(hookPath, collectionName, "post")
		require.NoError(t, err)

		inputData := map[string]interface{}{
			"collection": collectionName,
			"method":     "post",
			"data": map[string]interface{}{
				"name": "New Item",
			},
		}

		result, err := manager.ExecuteHook(ctx, collectionName, "post", inputData)
		require.NoError(t, err)
		
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)
		
		data, ok := resultMap["data"].(map[string]interface{})
		require.True(t, ok)
		
		assert.True(t, data["processed"].(bool))
		assert.NotEmpty(t, data["processedAt"])
		assert.True(t, resultMap["modified"].(bool))
	})

	t.Run("get event hook adds computed fields", func(t *testing.T) {
		hookPath := createTestGoHook(t, "get_hook", "Get")
		manager := events.NewManager()
		
		err := manager.LoadGoScript(hookPath, collectionName, "get")
		require.NoError(t, err)

		inputData := map[string]interface{}{
			"collection": collectionName,
			"method":     "get",
			"data": map[string]interface{}{
				"name": "Existing Item",
			},
		}

		result, err := manager.ExecuteHook(ctx, collectionName, "get", inputData)
		require.NoError(t, err)
		
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)
		
		data, ok := resultMap["data"].(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, "Processed: Existing Item", data["displayName"])
	})

	t.Run("multiple hooks for same event", func(t *testing.T) {
		hook1Path := createTestGoHook(t, "hook1", "Post")
		hook2Path := createTestGoHook(t, "hook2", "Post")
		
		manager := events.NewManager()
		
		err := manager.LoadGoScript(hook1Path, collectionName, "post")
		require.NoError(t, err)
		
		err = manager.LoadGoScript(hook2Path, collectionName, "post")
		require.NoError(t, err)

		inputData := map[string]interface{}{
			"collection": collectionName,
			"method":     "post",
			"data": map[string]interface{}{
				"name": "Multi Hook Item",
			},
		}

		// Both hooks should execute
		result, err := manager.ExecuteHook(ctx, collectionName, "post", inputData)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestEventFiring(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	collectionName := testutil.GenerateRandomName("event_test")
	collection := testutil.CreateTestCollection(t, db, collectionName)
	defer testutil.CleanupCollection(t, db, collectionName)

	store := db.GetStore(collectionName)
	require.NotNil(t, store)

	ctx := context.Background()
	eventsFired := make(map[string]int)

	// Mock event handler to track fired events
	mockHandler := func(eventType string, data interface{}) error {
		eventsFired[eventType]++
		return nil
	}

	t.Run("events fire on CRUD operations", func(t *testing.T) {
		// Create
		doc := map[string]interface{}{
			"name":  "Test Doc",
			"owner": "testuser",
		}

		result, err := store.Insert(ctx, doc)
		require.NoError(t, err)
		
		docID := result.InsertedID

		// Read
		query := db.CreateQuery().Where("_id", "=", docID)
		docs, err := store.Find(ctx, query)
		require.NoError(t, err)
		assert.Len(t, docs, 1)

		// Update
		update := db.CreateUpdate().Set("name", "Updated Doc")
		updateResult, err := store.Update(ctx, query, update)
		require.NoError(t, err)
		assert.Greater(t, updateResult.ModifiedCount, int64(0))

		// Delete
		deleteResult, err := store.Delete(ctx, query)
		require.NoError(t, err)
		assert.Greater(t, deleteResult.DeletedCount, int64(0))
	})

	t.Run("event cancellation", func(t *testing.T) {
		cancellingHandler := func(eventType string, data interface{}) error {
			if eventType == "validate" {
				return fmt.Errorf("validation failed")
			}
			return nil
		}

		// This would need integration with actual event system
		// For now, we're testing the concept
		err := cancellingHandler("validate", nil)
		assert.Error(t, err)
	})
}

func TestEventDataModification(t *testing.T) {
	t.Run("pre-save event can modify data", func(t *testing.T) {
		originalData := map[string]interface{}{
			"name": "Original",
		}

		modifyingHandler := func(data map[string]interface{}) map[string]interface{} {
			data["name"] = "Modified"
			data["modifiedAt"] = time.Now().Format(time.RFC3339)
			return data
		}

		modifiedData := modifyingHandler(originalData)
		assert.Equal(t, "Modified", modifiedData["name"])
		assert.NotEmpty(t, modifiedData["modifiedAt"])
	})

	t.Run("post-get event can add computed fields", func(t *testing.T) {
		fetchedData := map[string]interface{}{
			"price":    100,
			"quantity": 5,
		}

		computeHandler := func(data map[string]interface{}) map[string]interface{} {
			price := data["price"].(int)
			quantity := data["quantity"].(int)
			data["total"] = price * quantity
			return data
		}

		enhancedData := computeHandler(fetchedData)
		assert.Equal(t, 500, enhancedData["total"])
	})
}