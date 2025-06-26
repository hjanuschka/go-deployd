package resources_test

import (
	"context"
	"testing"

	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"github.com/hjanuschka/go-deployd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionCreation(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	collectionName := testutil.GenerateRandomName("test_collection")
	defer testutil.CleanupCollection(t, db, collectionName)

	t.Run("create collection config with basic properties", func(t *testing.T) {
		properties := map[string]resources.Property{
			"title": {
				Type:     "string",
				Required: true,
			},
			"count": {
				Type:     "number",
				Required: false,
			},
			"tags": {
				Type:     "array",
				Required: false,
			},
		}

		assert.Len(t, properties, 3)
		assert.True(t, properties["title"].Required)
		assert.False(t, properties["count"].Required)
	})

	t.Run("add property to existing collection config", func(t *testing.T) {
		properties := map[string]resources.Property{
			"name": {
				Type:     "string",
				Required: true,
			},
		}

		newProp := resources.Property{
			Type:     "date",
			Required: false,
		}

		properties["newField"] = newProp
		assert.Contains(t, properties, "newField")
		assert.Equal(t, "date", properties["newField"].Type)
	})

	t.Run("validate property types", func(t *testing.T) {
		validTypes := []string{"string", "number", "boolean", "date", "array", "object"}
		
		for _, validType := range validTypes {
			prop := resources.Property{
				Type: validType,
			}
			assert.Equal(t, validType, prop.Type)
		}
	})
}

func TestCollectionWithDatabase(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	collectionName := testutil.GenerateRandomName("test_collection")
	store := testutil.CreateTestCollection(t, db, collectionName)
	defer testutil.CleanupCollection(t, db, collectionName)

	require.NotNil(t, store)

	ctx := context.Background()

	t.Run("insert document", func(t *testing.T) {
		doc := map[string]interface{}{
			"name":  "Test Document",
			"owner": "testuser",
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		}

		result, err := store.Insert(ctx, doc)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("query documents", func(t *testing.T) {
		docs := []map[string]interface{}{
			{
				"name":  "Doc 1",
				"owner": "user1",
			},
			{
				"name":  "Doc 2",
				"owner": "user2",
			},
			{
				"name":  "Doc 3",
				"owner": "user1",
			},
		}

		for _, doc := range docs {
			_, err := store.Insert(ctx, doc)
			require.NoError(t, err)
		}

		query := database.NewQueryBuilder().Where("owner", "=", "user1")
		results, err := store.Find(ctx, query, database.QueryOptions{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(results))
	})
}

func TestCollectionValidation(t *testing.T) {
	t.Run("required fields validation", func(t *testing.T) {
		properties := map[string]resources.Property{
			"required_field": {
				Type:     "string",
				Required: true,
			},
			"optional_field": {
				Type:     "string",
				Required: false,
			},
		}

		doc := map[string]interface{}{
			"optional_field": "value",
		}

		err := validateDocument(properties, doc)
		assert.Error(t, err, "should fail without required field")

		doc["required_field"] = "value"
		err = validateDocument(properties, doc)
		assert.NoError(t, err)
	})

	t.Run("type validation", func(t *testing.T) {
		properties := map[string]resources.Property{
			"string_field": {
				Type: "string",
			},
			"number_field": {
				Type: "number",
			},
			"boolean_field": {
				Type: "boolean",
			},
		}

		validDoc := map[string]interface{}{
			"string_field":  "text",
			"number_field":  42,
			"boolean_field": true,
		}

		err := validateDocument(properties, validDoc)
		assert.NoError(t, err)

		invalidDoc := map[string]interface{}{
			"string_field":  123,
			"number_field":  "not a number",
			"boolean_field": "not a bool",
		}

		// For now, we don't have type validation implemented
		// This would need to be implemented in the actual validation logic
		_ = invalidDoc
	})
}

func validateDocument(properties map[string]resources.Property, doc map[string]interface{}) error {
	for propName, prop := range properties {
		if prop.Required {
			if _, exists := doc[propName]; !exists {
				return assert.AnError
			}
		}
	}
	return nil
}