package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSimpleQueryTranslation tests MongoDB-to-SQL translation with simpler verification
// This focuses on ensuring the translation logic works without getting into specifics
// of whether fields use columns or JSON extraction
func TestSimpleQueryTranslation(t *testing.T) {
	tests := []struct {
		name           string
		mongoQuery     map[string]interface{}
		expectContains []string // Strings that should appear in the SQL
		expectArgs     int      // Expected number of arguments
		description    string
	}{
		{
			name: "Simple equality",
			mongoQuery: map[string]interface{}{
				"title": "test",
			},
			expectContains: []string{"=", "?"},
			expectArgs:     1,
			description:    "Basic field equality should use = operator",
		},
		{
			name: "Greater than operator",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gt": 5,
				},
			},
			expectContains: []string{">", "?"},
			expectArgs:     1,
			description:    "MongoDB $gt should translate to SQL >",
		},
		{
			name: "Greater than or equal",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gte": 5,
				},
			},
			expectContains: []string{">=", "?"},
			expectArgs:     1,
			description:    "MongoDB $gte should translate to SQL >=",
		},
		{
			name: "Less than operator",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$lt": 10,
				},
			},
			expectContains: []string{"<", "?"},
			expectArgs:     1,
			description:    "MongoDB $lt should translate to SQL <",
		},
		{
			name: "Less than or equal",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$lte": 10,
				},
			},
			expectContains: []string{"<=", "?"},
			expectArgs:     1,
			description:    "MongoDB $lte should translate to SQL <=",
		},
		{
			name: "Not equal operator",
			mongoQuery: map[string]interface{}{
				"status": map[string]interface{}{
					"$ne": "deleted",
				},
			},
			expectContains: []string{"!=", "?"},
			expectArgs:     1,
			description:    "MongoDB $ne should translate to SQL !=",
		},
		{
			name: "IN operator",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$in": []interface{}{1, 2, 3},
				},
			},
			expectContains: []string{"IN", "(", "?"},
			expectArgs:     3,
			description:    "MongoDB $in should translate to SQL IN",
		},
		{
			name: "NOT IN operator",
			mongoQuery: map[string]interface{}{
				"status": map[string]interface{}{
					"$nin": []interface{}{"deleted", "archived"},
				},
			},
			expectContains: []string{"NOT", "IN", "(", "?"},
			expectArgs:     2,
			description:    "MongoDB $nin should translate to SQL NOT IN",
		},
		{
			name: "REGEX operator",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test",
				},
			},
			expectContains: []string{"LIKE", "?"},
			expectArgs:     1,
			description:    "MongoDB $regex should translate to SQL LIKE",
		},
		{
			name: "EXISTS operator true",
			mongoQuery: map[string]interface{}{
				"metadata": map[string]interface{}{
					"$exists": true,
				},
			},
			expectContains: []string{"IS NOT NULL"},
			expectArgs:     0,
			description:    "MongoDB $exists true should translate to IS NOT NULL",
		},
		{
			name: "EXISTS operator false",
			mongoQuery: map[string]interface{}{
				"metadata": map[string]interface{}{
					"$exists": false,
				},
			},
			expectContains: []string{"IS NULL"},
			expectArgs:     0,
			description:    "MongoDB $exists false should translate to IS NULL",
		},
		{
			name: "Multiple conditions",
			mongoQuery: map[string]interface{}{
				"title": "test",
				"priority": map[string]interface{}{
					"$gte": 5,
				},
			},
			expectContains: []string{"=", ">=", "AND"},
			expectArgs:     2,
			description:    "Multiple conditions should be joined with AND",
		},
		{
			name: "Range query",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gte": 5,
					"$lte": 10,
				},
			},
			expectContains: []string{">=", "<=", "AND"},
			expectArgs:     2,
			description:    "Range queries should generate multiple conditions",
		},
		{
			name: "OR conditions",
			mongoQuery: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"title": "test1"},
					map[string]interface{}{"title": "test2"},
				},
			},
			expectContains: []string{"(", "OR", ")"},
			expectArgs:     2,
			description:    "MongoDB $or should translate to SQL OR in parentheses",
		},
		{
			name: "AND conditions",
			mongoQuery: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"completed": false},
					map[string]interface{}{
						"priority": map[string]interface{}{
							"$gt": 5,
						},
					},
				},
			},
			expectContains: []string{"(", "AND", ")"},
			expectArgs:     2,
			description:    "MongoDB $and should translate to SQL AND in parentheses",
		},
		{
			name: "Complex nested query",
			mongoQuery: map[string]interface{}{
				"completed": false,
				"$or": []interface{}{
					map[string]interface{}{
						"title": map[string]interface{}{
							"$regex": "urgent",
						},
					},
					map[string]interface{}{
						"priority": map[string]interface{}{
							"$gte": 8,
						},
					},
				},
			},
			expectContains: []string{"=", "LIKE", ">=", "AND", "OR", "(", ")"},
			expectArgs:     3,
			description:    "Complex queries should maintain all operators and grouping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new SQL query builder
			builder := NewSQLQueryBuilder()
			
			// Convert the MongoDB query to SQL using the query builder
			convertMongoToSQLQuery(tt.mongoQuery, builder)
			
			// Get the generated SQL and arguments
			actualSQL, actualArgs := builder.ToSQL()
			
			// Check that expected strings appear in the SQL
			for _, expected := range tt.expectContains {
				assert.Contains(t, actualSQL, expected, 
					"SQL should contain '%s' for %s: %s", expected, tt.name, tt.description)
			}
			
			// Check argument count
			assert.Equal(t, tt.expectArgs, len(actualArgs), 
				"Expected %d arguments for %s: %s", tt.expectArgs, tt.name, tt.description)
			
			// Log for debugging
			t.Logf("Test: %s", tt.description)
			t.Logf("MongoDB: %+v", tt.mongoQuery)
			t.Logf("Generated SQL: %s", actualSQL)
			t.Logf("Arguments: %+v", actualArgs)
		})
	}
}

// TestRegexPatternConversionSimple tests the regex pattern conversion
func TestRegexPatternConversionSimple(t *testing.T) {
	tests := []struct {
		name        string
		mongoQuery  map[string]interface{}
		expectLike  bool
		description string
	}{
		{
			name: "Simple regex",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test",
				},
			},
			expectLike:  true,
			description: "Simple regex should generate LIKE clause",
		},
		{
			name: "Anchored regex",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "^test$",
				},
			},
			expectLike:  true,
			description: "Anchored regex should still generate LIKE clause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewSQLQueryBuilder()
			convertMongoToSQLQuery(tt.mongoQuery, builder)
			sql, args := builder.ToSQL()
			
			if tt.expectLike {
				assert.Contains(t, sql, "LIKE", "Should contain LIKE operator")
				assert.Greater(t, len(args), 0, "Should have at least one argument")
			}
			
			t.Logf("Test: %s", tt.description)
			t.Logf("SQL: %s", sql)
			t.Logf("Args: %+v", args)
		})
	}
}

// TestQueryValidation tests that queries are properly validated and don't break
func TestQueryValidation(t *testing.T) {
	tests := []struct {
		name        string
		mongoQuery  map[string]interface{}
		shouldPanic bool
		description string
	}{
		{
			name:        "Empty query",
			mongoQuery:  map[string]interface{}{},
			shouldPanic: false,
			description: "Empty query should not panic",
		},
		{
			name: "Nil operator value",
			mongoQuery: map[string]interface{}{
				"field": map[string]interface{}{
					"$gt": nil,
				},
			},
			shouldPanic: false,
			description: "Nil operator values should not panic",
		},
		{
			name: "Empty OR array",
			mongoQuery: map[string]interface{}{
				"$or": []interface{}{},
			},
			shouldPanic: false,
			description: "Empty OR array should not panic",
		},
		{
			name: "Invalid OR structure",
			mongoQuery: map[string]interface{}{
				"$or": "invalid",
			},
			shouldPanic: false,
			description: "Invalid OR structure should not panic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(t, func() {
					builder := NewSQLQueryBuilder()
					convertMongoToSQLQuery(tt.mongoQuery, builder)
					builder.ToSQL()
				}, "Should panic for: %s", tt.description)
			} else {
				assert.NotPanics(t, func() {
					builder := NewSQLQueryBuilder()
					convertMongoToSQLQuery(tt.mongoQuery, builder)
					sql, args := builder.ToSQL()
					t.Logf("Query: %+v -> SQL: %s, Args: %+v", tt.mongoQuery, sql, args)
				}, "Should not panic for: %s", tt.description)
			}
		})
	}
}

// TestOperatorSupport verifies that all supported MongoDB operators work
func TestOperatorSupport(t *testing.T) {
	supportedOps := []string{
		"$eq", "$ne", "$gt", "$gte", "$lt", "$lte",
		"$in", "$nin", "$regex", "$exists",
	}

	for _, op := range supportedOps {
		t.Run("Operator_"+op, func(t *testing.T) {
			var mongoQuery map[string]interface{}
			
			switch op {
			case "$in", "$nin":
				mongoQuery = map[string]interface{}{
					"field": map[string]interface{}{
						op: []interface{}{"value1", "value2"},
					},
				}
			case "$exists":
				mongoQuery = map[string]interface{}{
					"field": map[string]interface{}{
						op: true,
					},
				}
			default:
				mongoQuery = map[string]interface{}{
					"field": map[string]interface{}{
						op: "value",
					},
				}
			}
			
			// Should not panic and should generate some SQL
			assert.NotPanics(t, func() {
				builder := NewSQLQueryBuilder()
				convertMongoToSQLQuery(mongoQuery, builder)
				sql, args := builder.ToSQL()
				
				// Should generate non-empty SQL for most operators
				if op != "$exists" {
					assert.NotEmpty(t, sql, "Should generate SQL for operator %s", op)
				}
				
				t.Logf("Operator %s: SQL=%s, Args=%+v", op, sql, args)
			})
		})
	}
}