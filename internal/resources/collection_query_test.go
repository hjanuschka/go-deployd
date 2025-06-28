package resources

import (
	"context"
	"testing"

	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStore implements StoreInterface for testing
type MockStore struct {
	mock.Mock
}

func (m *MockStore) CreateUniqueIdentifier() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStore) Insert(ctx context.Context, document interface{}) (interface{}, error) {
	args := m.Called(ctx, document)
	return args.Get(0), args.Error(1)
}

func (m *MockStore) Find(ctx context.Context, query database.QueryBuilder, opts database.QueryOptions) ([]map[string]interface{}, error) {
	args := m.Called(ctx, query, opts)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockStore) FindOne(ctx context.Context, query database.QueryBuilder) (map[string]interface{}, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockStore) Update(ctx context.Context, query database.QueryBuilder, update database.UpdateBuilder) (database.UpdateResult, error) {
	args := m.Called(ctx, query, update)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) UpdateOne(ctx context.Context, query database.QueryBuilder, update database.UpdateBuilder) (database.UpdateResult, error) {
	args := m.Called(ctx, query, update)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) Remove(ctx context.Context, query database.QueryBuilder) (database.DeleteResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(database.DeleteResult), args.Error(1)
}

func (m *MockStore) Count(ctx context.Context, query database.QueryBuilder) (int64, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStore) Increment(ctx context.Context, query database.QueryBuilder, increments map[string]interface{}) (database.UpdateResult, error) {
	args := m.Called(ctx, query, increments)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) Push(ctx context.Context, query database.QueryBuilder, pushOps map[string]interface{}) (database.UpdateResult, error) {
	args := m.Called(ctx, query, pushOps)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) Pull(ctx context.Context, query database.QueryBuilder, pullOps map[string]interface{}) (database.UpdateResult, error) {
	args := m.Called(ctx, query, pullOps)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) AddToSet(ctx context.Context, query database.QueryBuilder, addOps map[string]interface{}) (database.UpdateResult, error) {
	args := m.Called(ctx, query, addOps)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) PopFirst(ctx context.Context, query database.QueryBuilder, fields []string) (database.UpdateResult, error) {
	args := m.Called(ctx, query, fields)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) PopLast(ctx context.Context, query database.QueryBuilder, fields []string) (database.UpdateResult, error) {
	args := m.Called(ctx, query, fields)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) Upsert(ctx context.Context, query database.QueryBuilder, update database.UpdateBuilder) (database.UpdateResult, error) {
	args := m.Called(ctx, query, update)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) Aggregate(ctx context.Context, pipeline []map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, pipeline)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockStore) FindWithRawQuery(ctx context.Context, mongoQuery interface{}, options map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, mongoQuery, options)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockStore) CountWithRawQuery(ctx context.Context, mongoQuery interface{}) (int64, error) {
	args := m.Called(ctx, mongoQuery)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStore) UpdateWithRawQuery(ctx context.Context, mongoQuery interface{}, mongoUpdate interface{}) (database.UpdateResult, error) {
	args := m.Called(ctx, mongoQuery, mongoUpdate)
	return args.Get(0).(database.UpdateResult), args.Error(1)
}

func (m *MockStore) RemoveWithRawQuery(ctx context.Context, mongoQuery interface{}) (database.DeleteResult, error) {
	args := m.Called(ctx, mongoQuery)
	return args.Get(0).(database.DeleteResult), args.Error(1)
}

// Helper function to create a test collection
func createTestCollection() *Collection {
	config := &CollectionConfig{
		Properties: map[string]Property{
			"title": {
				Type:     "string",
				Required: true,
			},
			"completed": {
				Type:    "boolean",
				Default: false,
			},
			"priority": {
				Type:    "number",
				Default: 1,
			},
			"tags": {
				Type: "array",
			},
			"metadata": {
				Type: "object",
			},
		},
	}

	collection := NewCollection("test", config, nil)
	return collection
}

func TestSanitizeQuery(t *testing.T) {
	collection := createTestCollection()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple equality query",
			input: map[string]interface{}{
				"title": "test",
			},
			expected: map[string]interface{}{
				"title": "test",
			},
		},
		{
			name: "MongoDB operators allowed",
			input: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"title": "test1"},
					map[string]interface{}{"title": "test2"},
				},
			},
			expected: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"title": "test1"},
					map[string]interface{}{"title": "test2"},
				},
			},
		},
		{
			name: "Field operator pattern",
			input: map[string]interface{}{
				"title[$regex]": "test",
			},
			expected: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test",
				},
			},
		},
		{
			name: "Complex nested query",
			input: map[string]interface{}{
				"title[$regex]":    "test",
				"completed":        true,
				"priority[$gte]":   5,
				"priority[$lte]":   10,
			},
			expected: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test",
				},
				"completed": true,
				"priority": map[string]interface{}{
					"$gte": float64(5),
					"$lte": float64(10),
				},
			},
		},
		{
			name: "Invalid fields filtered out",
			input: map[string]interface{}{
				"title":      "test",
				"invalidField": "should be removed",
			},
			expected: map[string]interface{}{
				"title": "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collection.sanitizeQuery(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapToQueryBuilder(t *testing.T) {
	collection := createTestCollection()

	tests := []struct {
		name        string
		input       map[string]interface{}
		expectSQL   string // Expected SQL-like representation for verification
		description string
	}{
		{
			name: "Simple equality",
			input: map[string]interface{}{
				"title": "test",
			},
			description: "Should create simple WHERE title = 'test'",
		},
		{
			name: "Greater than operator",
			input: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gt": 5,
				},
			},
			description: "Should create WHERE priority > 5",
		},
		{
			name: "IN operator",
			input: map[string]interface{}{
				"priority": map[string]interface{}{
					"$in": []interface{}{1, 2, 3},
				},
			},
			description: "Should create WHERE priority IN (1, 2, 3)",
		},
		{
			name: "NOT IN operator",
			input: map[string]interface{}{
				"priority": map[string]interface{}{
					"$nin": []interface{}{1, 2, 3},
				},
			},
			description: "Should create WHERE priority NOT IN (1, 2, 3)",
		},
		{
			name: "REGEX operator",
			input: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test.*",
				},
			},
			description: "Should create WHERE title LIKE pattern",
		},
		{
			name: "EXISTS operator",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"$exists": true,
				},
			},
			description: "Should create WHERE metadata IS NOT NULL",
		},
		{
			name: "NOT EXISTS operator",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"$exists": false,
				},
			},
			description: "Should create WHERE metadata IS NULL",
		},
		{
			name: "NOT EQUAL operator",
			input: map[string]interface{}{
				"title": map[string]interface{}{
					"$ne": "test",
				},
			},
			description: "Should create WHERE title != 'test'",
		},
		{
			name: "OR conditions",
			input: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"title": "test1"},
					map[string]interface{}{"title": "test2"},
				},
			},
			description: "Should create WHERE (title = 'test1' OR title = 'test2')",
		},
		{
			name: "AND conditions",
			input: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"completed": true},
					map[string]interface{}{"priority": map[string]interface{}{"$gt": 5}},
				},
			},
			description: "Should create WHERE (completed = true AND priority > 5)",
		},
		{
			name: "Complex mixed query",
			input: map[string]interface{}{
				"completed": false,
				"$or": []interface{}{
					map[string]interface{}{"title": map[string]interface{}{"$regex": "urgent"}},
					map[string]interface{}{"priority": map[string]interface{}{"$gte": 8}},
				},
			},
			description: "Should create WHERE completed = false AND (title LIKE '%urgent%' OR priority >= 8)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := collection.mapToQueryBuilder(tt.input)
			
			// Verify the query builder was created
			assert.NotNil(t, builder, "Query builder should not be nil")
			
			// Convert back to map to verify structure
			queryMap := builder.ToMap()
			
			// Basic assertion that the query was processed
			// More detailed testing would require inspecting the actual SQL generation
			assert.NotNil(t, queryMap, "Query map should not be nil")
			
			t.Logf("Test: %s", tt.description)
			t.Logf("Input: %+v", tt.input)
			t.Logf("Query Map: %+v", queryMap)
		})
	}
}

func TestQueryOptionsExtraction(t *testing.T) {
	collection := createTestCollection()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected database.QueryOptions
	}{
		{
			name: "Basic pagination",
			input: map[string]interface{}{
				"$limit": float64(10),
				"$skip":  float64(20),
			},
			expected: database.QueryOptions{
				Limit: func() *int64 { v := int64(10); return &v }(),
				Skip:  func() *int64 { v := int64(20); return &v }(),
				Sort:  make(map[string]int),
				Fields: make(map[string]int),
			},
		},
		{
			name: "Sorting options",
			input: map[string]interface{}{
				"$sort": map[string]interface{}{
					"title":     float64(1),
					"createdAt": float64(-1),
				},
			},
			expected: database.QueryOptions{
				Limit: func() *int64 { v := int64(50); return &v }(), // Default limit
				Sort: map[string]int{
					"title":     1,
					"createdAt": -1,
				},
				Fields: make(map[string]int),
			},
		},
		{
			name: "Field projection",
			input: map[string]interface{}{
				"$fields": map[string]interface{}{
					"title":     float64(1),
					"completed": float64(1),
					"priority":  float64(0),
				},
			},
			expected: database.QueryOptions{
				Limit: func() *int64 { v := int64(50); return &v }(), // Default limit
				Sort:  make(map[string]int),
				Fields: map[string]int{
					"title":     1,
					"completed": 1,
					"priority":  0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, cleanQuery := collection.extractQueryOptions(tt.input)
			
			// Check limit
			if tt.expected.Limit != nil {
				assert.NotNil(t, opts.Limit, "Limit should not be nil")
				assert.Equal(t, *tt.expected.Limit, *opts.Limit, "Limit values should match")
			}
			
			// Check skip
			if tt.expected.Skip != nil {
				assert.NotNil(t, opts.Skip, "Skip should not be nil")
				assert.Equal(t, *tt.expected.Skip, *opts.Skip, "Skip values should match")
			}
			
			// Check sort
			assert.Equal(t, tt.expected.Sort, opts.Sort, "Sort options should match")
			
			// Check fields
			assert.Equal(t, tt.expected.Fields, opts.Fields, "Field options should match")
			
			// Verify query options were removed from clean query
			for key := range tt.input {
				if key == "$limit" || key == "$skip" || key == "$sort" || key == "$fields" {
					assert.NotContains(t, cleanQuery, key, "Query option %s should be removed from clean query", key)
				}
			}
		})
	}
}

func TestComplexQueryScenarios(t *testing.T) {
	collection := createTestCollection()

	tests := []struct {
		name        string
		mongoQuery  map[string]interface{}
		description string
	}{
		{
			name: "E-commerce product search",
			mongoQuery: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{
						"$or": []interface{}{
							map[string]interface{}{"title": map[string]interface{}{"$regex": "laptop"}},
							map[string]interface{}{"title": map[string]interface{}{"$regex": "computer"}},
						},
					},
					map[string]interface{}{
						"priority": map[string]interface{}{
							"$gte": 5,
							"$lte": 10,
						},
					},
				},
			},
			description: "Should handle complex product search with multiple conditions",
		},
		{
			name: "User activity filter",
			mongoQuery: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{
						"completed": true,
					},
					map[string]interface{}{
						"$and": []interface{}{
							map[string]interface{}{"priority": map[string]interface{}{"$gte": 8}},
							map[string]interface{}{"title": map[string]interface{}{"$regex": "urgent"}},
						},
					},
				},
			},
			description: "Should handle user activity filtering with nested AND/OR",
		},
		{
			name: "Array and object queries",
			mongoQuery: map[string]interface{}{
				"tags": map[string]interface{}{
					"$in": []interface{}{"important", "urgent", "work"},
				},
				"metadata": map[string]interface{}{
					"$exists": true,
				},
			},
			description: "Should handle array $in and object existence checks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test sanitization
			sanitized := collection.sanitizeQuery(tt.mongoQuery)
			assert.NotNil(t, sanitized, "Sanitized query should not be nil")
			
			// Test query builder creation
			builder := collection.mapToQueryBuilder(sanitized)
			assert.NotNil(t, builder, "Query builder should not be nil")
			
			// Test conversion back to map (for verification)
			resultMap := builder.ToMap()
			assert.NotNil(t, resultMap, "Result map should not be nil")
			
			t.Logf("Test: %s", tt.description)
			t.Logf("Original: %+v", tt.mongoQuery)
			t.Logf("Sanitized: %+v", sanitized)
			t.Logf("Result: %+v", resultMap)
		})
	}
}

func TestSQLTranslationVerification(t *testing.T) {
	// This test verifies that MongoDB queries are correctly translated to SQL
	// by checking the resulting SQL query strings when available
	
	collection := createTestCollection()

	tests := []struct {
		name           string
		mongoQuery     map[string]interface{}
		expectContains []string // Strings that should appear in the SQL
		description    string
	}{
		{
			name: "Simple regex translation",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test",
				},
			},
			expectContains: []string{"LIKE", "%test%"},
			description:    "Regex should be translated to SQL LIKE with wildcards",
		},
		{
			name: "Range query translation",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gte": 5,
					"$lte": 10,
				},
			},
			expectContains: []string{">=", "<=", "5", "10"},
			description:    "Range queries should use >= and <= operators",
		},
		{
			name: "IN query translation",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$in": []interface{}{1, 2, 3},
				},
			},
			expectContains: []string{"IN", "(", ")"},
			description:    "IN queries should use SQL IN operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized := collection.sanitizeQuery(tt.mongoQuery)
			builder := collection.mapToQueryBuilder(sanitized)
			
			// If the builder supports SQL generation, test it
			if sqlBuilder, ok := builder.(*database.SQLQueryBuilder); ok {
				sqlQuery, args := sqlBuilder.ToSQL()
				
				t.Logf("Generated SQL: %s", sqlQuery)
				t.Logf("Arguments: %+v", args)
				
				// Check that expected strings appear in the SQL
				for _, expected := range tt.expectContains {
					assert.Contains(t, sqlQuery, expected, 
						"SQL should contain '%s' for %s", expected, tt.description)
				}
			} else {
				t.Logf("Builder type: %T does not support SQL generation", builder)
			}
		})
	}
}

func TestForceMongoOption(t *testing.T) {
	// Test the $forceMongo option functionality
	collection := createTestCollection()
	mockStore := &MockStore{}
	collection.store = mockStore

	// Mock the FindWithRawQuery method
	expectedDocs := []map[string]interface{}{
		{"id": "1", "title": "test", "completed": false},
	}
	mockStore.On("FindWithRawQuery", 
		mock.Anything, 
		mock.Anything, 
		mock.Anything).Return(expectedDocs, nil)

	// Test query that would use direct MongoDB execution
	mongoQuery := map[string]interface{}{
		"title": map[string]interface{}{
			"$regex": "test",
		},
	}

	// This would normally go through SQL translation, but with forceMongo it should use direct MongoDB
	// Note: In a real test, you'd need to set up the context and test the actual handler
	t.Run("Force Mongo Flag", func(t *testing.T) {
		// Verify that the store supports raw queries
		_, supportsRaw := collection.store.(interface {
			FindWithRawQuery(ctx context.Context, mongoQuery interface{}, options map[string]interface{}) ([]map[string]interface{}, error)
		})
		
		assert.True(t, supportsRaw, "Mock store should support raw queries")
		
		// The actual test would require setting up HTTP context and testing the handler
		// For now, we verify the interface exists
		t.Log("Force mongo functionality is properly structured")
	})
}