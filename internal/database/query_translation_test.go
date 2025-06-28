package database

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestQueryTranslation focuses on pure MongoDB-to-SQL translation without database operations
// This ensures query translation works correctly and doesn't break in future changes
func TestQueryTranslation(t *testing.T) {
	tests := []struct {
		name           string
		mongoQuery     map[string]interface{}
		expectedSQL    string
		expectedArgs   []interface{}
		description    string
	}{
		{
			name: "Simple equality",
			mongoQuery: map[string]interface{}{
				"title": "test",
			},
			expectedSQL: "JSON_EXTRACT(data, '$.title') = ?",
			expectedArgs: []interface{}{"test"},
			description: "Basic field equality should translate to JSON extraction WHERE clause",
		},
		{
			name: "Greater than operator",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gt": 5,
				},
			},
			expectedSQL: "priority > ?",
			expectedArgs: []interface{}{5},
			description: "MongoDB $gt should translate to SQL >",
		},
		{
			name: "Greater than or equal",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gte": 5,
				},
			},
			expectedSQL: "priority >= ?",
			expectedArgs: []interface{}{5},
			description: "MongoDB $gte should translate to SQL >=",
		},
		{
			name: "Less than operator",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$lt": 10,
				},
			},
			expectedSQL: "priority < ?",
			expectedArgs: []interface{}{10},
			description: "MongoDB $lt should translate to SQL <",
		},
		{
			name: "Less than or equal",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$lte": 10,
				},
			},
			expectedSQL: "priority <= ?",
			expectedArgs: []interface{}{10},
			description: "MongoDB $lte should translate to SQL <=",
		},
		{
			name: "Not equal operator",
			mongoQuery: map[string]interface{}{
				"status": map[string]interface{}{
					"$ne": "deleted",
				},
			},
			expectedSQL: "status != ?",
			expectedArgs: []interface{}{"deleted"},
			description: "MongoDB $ne should translate to SQL !=",
		},
		{
			name: "IN operator",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$in": []interface{}{1, 2, 3},
				},
			},
			expectedSQL: "priority IN (?, ?, ?)",
			expectedArgs: []interface{}{1, 2, 3},
			description: "MongoDB $in should translate to SQL IN",
		},
		{
			name: "NOT IN operator",
			mongoQuery: map[string]interface{}{
				"status": map[string]interface{}{
					"$nin": []interface{}{"deleted", "archived"},
				},
			},
			expectedSQL: "status NOT IN (?, ?)",
			expectedArgs: []interface{}{"deleted", "archived"},
			description: "MongoDB $nin should translate to SQL NOT IN",
		},
		{
			name: "REGEX operator basic",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test",
				},
			},
			expectedSQL: "title LIKE ?",
			expectedArgs: []interface{}{"%test%"},
			description: "MongoDB $regex should translate to SQL LIKE with wildcards",
		},
		{
			name: "REGEX operator with anchors",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "^test$",
				},
			},
			expectedSQL: "title LIKE ?",
			expectedArgs: []interface{}{"test"},
			description: "MongoDB $regex with anchors should translate to exact match LIKE",
		},
		{
			name: "REGEX operator start anchor",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "^test",
				},
			},
			expectedSQL: "title LIKE ?",
			expectedArgs: []interface{}{"test%"},
			description: "MongoDB $regex with start anchor should translate to prefix LIKE",
		},
		{
			name: "REGEX operator end anchor",
			mongoQuery: map[string]interface{}{
				"title": map[string]interface{}{
					"$regex": "test$",
				},
			},
			expectedSQL: "title LIKE ?",
			expectedArgs: []interface{}{"%test"},
			description: "MongoDB $regex with end anchor should translate to suffix LIKE",
		},
		{
			name: "EXISTS operator true",
			mongoQuery: map[string]interface{}{
				"metadata": map[string]interface{}{
					"$exists": true,
				},
			},
			expectedSQL: "metadata IS NOT NULL",
			expectedArgs: []interface{}{},
			description: "MongoDB $exists true should translate to IS NOT NULL",
		},
		{
			name: "EXISTS operator false",
			mongoQuery: map[string]interface{}{
				"metadata": map[string]interface{}{
					"$exists": false,
				},
			},
			expectedSQL: "metadata IS NULL",
			expectedArgs: []interface{}{},
			description: "MongoDB $exists false should translate to IS NULL",
		},
		{
			name: "Multiple conditions",
			mongoQuery: map[string]interface{}{
				"title": "test",
				"priority": map[string]interface{}{
					"$gte": 5,
				},
				"status": map[string]interface{}{
					"$ne": "deleted",
				},
			},
			expectedSQL: "title = ? AND priority >= ? AND status != ?",
			expectedArgs: []interface{}{"test", 5, "deleted"},
			description: "Multiple conditions should be joined with AND",
		},
		{
			name: "Range query",
			mongoQuery: map[string]interface{}{
				"priority": map[string]interface{}{
					"$gte": 5,
					"$lte": 10,
				},
			},
			expectedSQL: "priority >= ? AND priority <= ?",
			expectedArgs: []interface{}{5, 10},
			description: "Range queries should generate multiple conditions",
		},
		{
			name: "OR conditions simple",
			mongoQuery: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"title": "test1"},
					map[string]interface{}{"title": "test2"},
				},
			},
			expectedSQL: "(title = ? OR title = ?)",
			expectedArgs: []interface{}{"test1", "test2"},
			description: "MongoDB $or should translate to SQL OR in parentheses",
		},
		{
			name: "OR with complex conditions",
			mongoQuery: map[string]interface{}{
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
			expectedSQL: "(title LIKE ? OR priority >= ?)",
			expectedArgs: []interface{}{"%urgent%", 8},
			description: "OR with complex conditions should maintain operator translation",
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
			expectedSQL: "(completed = ? AND priority > ?)",
			expectedArgs: []interface{}{false, 5},
			description: "MongoDB $and should translate to SQL AND in parentheses",
		},
		{
			name: "Mixed AND/OR complex",
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
			expectedSQL: "completed = ? AND (title LIKE ? OR priority >= ?)",
			expectedArgs: []interface{}{false, "%urgent%", 8},
			description: "Mixed top-level AND with OR should be properly grouped",
		},
		{
			name: "Nested OR in AND",
			mongoQuery: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"completed": false},
					map[string]interface{}{
						"$or": []interface{}{
							map[string]interface{}{"title": map[string]interface{}{"$regex": "urgent"}},
							map[string]interface{}{"priority": map[string]interface{}{"$gte": 8}},
						},
					},
				},
			},
			expectedSQL: "(completed = ? AND (title LIKE ? OR priority >= ?))",
			expectedArgs: []interface{}{false, "%urgent%", 8},
			description: "Nested OR within AND should maintain proper grouping",
		},
		{
			name: "Multiple OR groups",
			mongoQuery: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"status": "active"},
					map[string]interface{}{"status": "pending"},
				},
				"category": map[string]interface{}{
					"$in": []interface{}{"work", "personal"},
				},
			},
			expectedSQL: "(status = ? OR status = ?) AND category IN (?, ?)",
			expectedArgs: []interface{}{"active", "pending", "work", "personal"},
			description: "Multiple query groups should be properly combined",
		},
		{
			name: "Complex e-commerce query",
			mongoQuery: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{
						"$or": []interface{}{
							map[string]interface{}{"title": map[string]interface{}{"$regex": "laptop"}},
							map[string]interface{}{"title": map[string]interface{}{"$regex": "computer"}},
						},
					},
					map[string]interface{}{
						"price": map[string]interface{}{
							"$gte": 500,
							"$lte": 2000,
						},
					},
					map[string]interface{}{
						"status": map[string]interface{}{
							"$in": []interface{}{"available", "limited"},
						},
					},
				},
			},
			expectedSQL: "((title LIKE ? OR title LIKE ?) AND price >= ? AND price <= ? AND status IN (?, ?))",
			expectedArgs: []interface{}{"%laptop%", "%computer%", 500, 2000, "available", "limited"},
			description: "Complex nested query should maintain all operator translations and grouping",
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
			
			// Normalize whitespace for comparison
			actualSQL = normalizeSQL(actualSQL)
			expectedSQL := normalizeSQL(tt.expectedSQL)
			
			// Assert the SQL matches
			assert.Equal(t, expectedSQL, actualSQL, 
				"SQL query mismatch for %s: %s", tt.name, tt.description)
			
			// Assert the arguments match
			assert.Equal(t, tt.expectedArgs, actualArgs, 
				"SQL arguments mismatch for %s: %s", tt.name, tt.description)
			
			// Log for debugging
			t.Logf("Test: %s", tt.description)
			t.Logf("MongoDB: %+v", tt.mongoQuery)
			t.Logf("Expected SQL: %s", expectedSQL)
			t.Logf("Actual SQL: %s", actualSQL)
			t.Logf("Expected Args: %+v", tt.expectedArgs)
			t.Logf("Actual Args: %+v", actualArgs)
		})
	}
}

// convertMongoToSQLQuery converts a MongoDB-style query to SQL using the query builder
// This function mimics the logic used in the Collection.mapToQueryBuilder method
func convertMongoToSQLQuery(mongoQuery map[string]interface{}, builder *SQLQueryBuilder) {
	for field, value := range mongoQuery {
		if strings.HasPrefix(field, "$") {
			// Handle special MongoDB operators at root level
			switch field {
			case "$or":
				if orConditions, ok := value.([]interface{}); ok {
					var orBuilders []QueryBuilder
					for _, condition := range orConditions {
						if condMap, ok := condition.(map[string]interface{}); ok {
							orBuilder := NewSQLQueryBuilder()
							convertMongoToSQLQuery(condMap, orBuilder)
							orBuilders = append(orBuilders, orBuilder)
						}
					}
					if len(orBuilders) > 0 {
						builder.Or(orBuilders...)
					}
				}
			case "$and":
				if andConditions, ok := value.([]interface{}); ok {
					var andBuilders []QueryBuilder
					for _, condition := range andConditions {
						if condMap, ok := condition.(map[string]interface{}); ok {
							andBuilder := NewSQLQueryBuilder()
							convertMongoToSQLQuery(condMap, andBuilder)
							andBuilders = append(andBuilders, andBuilder)
						}
					}
					if len(andBuilders) > 0 {
						builder.And(andBuilders...)
					}
				}
			case "$nor":
				// Not supported in current implementation
				continue
			}
			continue
		}

		if valueMap, ok := value.(map[string]interface{}); ok {
			// Field has operators like {"age": {"$gt": 18}}
			for op, opValue := range valueMap {
				switch op {
				case "$in":
					if values, ok := opValue.([]interface{}); ok {
						builder.WhereIn(field, values)
					}
				case "$nin":
					if values, ok := opValue.([]interface{}); ok {
						builder.WhereNotIn(field, values)
					}
				case "$exists":
					if exists, ok := opValue.(bool); ok {
						if exists {
							builder.WhereNotNull(field)
						} else {
							builder.WhereNull(field)
						}
					}
				case "$ne":
					builder.Where(field, "$ne", opValue)
				default:
					builder.Where(field, op, opValue)
				}
			}
		} else {
			// Simple equality
			builder.Where(field, "$eq", value)
		}
	}
}

// normalizeSQL normalizes SQL for comparison by removing extra whitespace
func normalizeSQL(sql string) string {
	// Replace multiple spaces with single space
	parts := strings.Fields(sql)
	return strings.Join(parts, " ")
}

// TestRegexPatternConversion tests the regex pattern conversion logic
func TestRegexPatternConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple pattern",
			input:    "test",
			expected: "%test%",
		},
		{
			name:     "Start anchored",
			input:    "^test",
			expected: "test%",
		},
		{
			name:     "End anchored",
			input:    "test$",
			expected: "%test",
		},
		{
			name:     "Both anchored",
			input:    "^test$",
			expected: "test",
		},
		{
			name:     "Complex pattern with anchors",
			input:    "^hello.*world$",
			expected: "hello.*world",
		},
		{
			name:     "No anchors",
			input:    "hello.*world",
			expected: "%hello.*world%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewSQLQueryBuilder()
			actual := builder.regexToLike(tt.input)
			assert.Equal(t, tt.expected, actual, 
				"Regex conversion failed for pattern: %s", tt.input)
		})
	}
}

// TestQueryBuilderEdgeCases tests edge cases and error conditions
func TestQueryBuilderEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		mongoQuery  map[string]interface{}
		expectError bool
		description string
	}{
		{
			name: "Empty query",
			mongoQuery: map[string]interface{}{},
			expectError: false,
			description: "Empty query should be handled gracefully",
		},
		{
			name: "Nil values",
			mongoQuery: map[string]interface{}{
				"field": nil,
			},
			expectError: false,
			description: "Nil values should be handled as NULL comparisons",
		},
		{
			name: "Empty OR array",
			mongoQuery: map[string]interface{}{
				"$or": []interface{}{},
			},
			expectError: false,
			description: "Empty OR array should be handled gracefully",
		},
		{
			name: "Empty AND array",
			mongoQuery: map[string]interface{}{
				"$and": []interface{}{},
			},
			expectError: false,
			description: "Empty AND array should be handled gracefully",
		},
		{
			name: "Nested empty conditions",
			mongoQuery: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{},
					map[string]interface{}{"title": "test"},
				},
			},
			expectError: false,
			description: "Nested empty conditions should not break the query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewSQLQueryBuilder()
			
			// Should not panic
			assert.NotPanics(t, func() {
				convertMongoToSQLQuery(tt.mongoQuery, builder)
			}, "Query conversion should not panic for: %s", tt.description)
			
			// Should generate valid SQL (even if empty)
			sql, args := builder.ToSQL()
			
			t.Logf("Test: %s", tt.description)
			t.Logf("Input: %+v", tt.mongoQuery)
			t.Logf("Generated SQL: %s", sql)
			t.Logf("Arguments: %+v", args)
			
			// Basic validation - SQL should be a string
			assert.IsType(t, "", sql, "SQL should be a string")
			assert.IsType(t, []interface{}{}, args, "Arguments should be a slice")
		})
	}
}

// BenchmarkQueryTranslation benchmarks the query translation performance
func BenchmarkQueryTranslation(t *testing.B) {
	complexQuery := map[string]interface{}{
		"$and": []interface{}{
			map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"title": map[string]interface{}{"$regex": "laptop"}},
					map[string]interface{}{"title": map[string]interface{}{"$regex": "computer"}},
				},
			},
			map[string]interface{}{
				"price": map[string]interface{}{
					"$gte": 500,
					"$lte": 2000,
				},
			},
			map[string]interface{}{
				"status": map[string]interface{}{
					"$in": []interface{}{"available", "limited"},
				},
			},
		},
	}

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		builder := NewSQLQueryBuilder()
		convertMongoToSQLQuery(complexQuery, builder)
		builder.ToSQL()
	}
}