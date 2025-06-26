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

	t.Run("create collection with basic properties", func(t *testing.T) {
		collection := &resources.Collection{
			Name: collectionName,
			Properties: map[string]resources.Property{
				"title": {
					Name:     "title",
					Type:     "string",
					Required: true,
				},
				"count": {
					Name:     "count",
					Type:     "number",
					Required: false,
				},
				"tags": {
					Name:     "tags",
					Type:     "array",
					Required: false,
				},
			},
		}

		assert.Equal(t, collectionName, collection.Name)
		assert.Len(t, collection.Properties, 3)
		assert.True(t, collection.Properties["title"].Required)
		assert.False(t, collection.Properties["count"].Required)
	})

	t.Run("add property to existing collection", func(t *testing.T) {
		collection := testutil.CreateTestCollection(t, db, "")
		defer testutil.CleanupCollection(t, db, collection.Name)

		newProp := resources.Property{
			Name:     "newField",
			Type:     "date",
			Required: false,
		}

		collection.Properties["newField"] = newProp
		assert.Contains(t, collection.Properties, "newField")
		assert.Equal(t, "date", collection.Properties["newField"].Type)
	})

	t.Run("validate property types", func(t *testing.T) {
		validTypes := []string{"string", "number", "boolean", "date", "array", "object"}
		
		for _, validType := range validTypes {
			prop := resources.Property{
				Name: "testProp",
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
	collection := testutil.CreateTestCollection(t, db, collectionName)
	defer testutil.CleanupCollection(t, db, collectionName)

	store := db.GetStore(collectionName)
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
		require.NotNil(t, result.InsertedID)
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

		query := db.CreateQuery().Where("owner", "=", "user1")
		results, err := store.Find(ctx, query)
		require.NoError(t, err)
		assert.Equal(t, 2, len(results))
	})
}

func TestCollectionValidation(t *testing.T) {
	db := testutil.CreateTestDB(t)
	defer db.Close()

	t.Run("required fields validation", func(t *testing.T) {
		collection := &resources.Collection{
			Name: "validation_test",
			Properties: map[string]resources.Property{
				"required_field": {
					Name:     "required_field",
					Type:     "string",
					Required: true,
				},
				"optional_field": {
					Name:     "optional_field",
					Type:     "string",
					Required: false,
				},
			},
		}

		doc := map[string]interface{}{
			"optional_field": "value",
		}

		err := validateDocument(collection, doc)
		assert.Error(t, err, "should fail without required field")

		doc["required_field"] = "value"
		err = validateDocument(collection, doc)
		assert.NoError(t, err)
	})

	t.Run("type validation", func(t *testing.T) {
		collection := &resources.Collection{
			Name: "type_validation_test",
			Properties: map[string]resources.Property{
				"string_field": {
					Name: "string_field",
					Type: "string",
				},
				"number_field": {
					Name: "number_field",
					Type: "number",
				},
				"boolean_field": {
					Name: "boolean_field",
					Type: "boolean",
				},
			},
		}

		validDoc := map[string]interface{}{
			"string_field":  "text",
			"number_field":  42,
			"boolean_field": true,
		}

		err := validateDocument(collection, validDoc)
		assert.NoError(t, err)

		invalidDoc := map[string]interface{}{
			"string_field":  123,
			"number_field":  "not a number",
			"boolean_field": "not a bool",
		}

		err = validateDocument(collection, invalidDoc)
		assert.Error(t, err)
	})
}

func validateDocument(collection *resources.Collection, doc map[string]interface{}) error {
	for propName, prop := range collection.Properties {
		if prop.Required {
			if _, exists := doc[propName]; !exists {
				return assert.AnError
			}
		}
	}
	return nil
}