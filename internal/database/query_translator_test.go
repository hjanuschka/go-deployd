package database

import (
	"testing"
	"reflect"
)

func TestQueryTranslator_SimpleQueries(t *testing.T) {
	tests := []struct {
		name     string
		dialect  string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name:     "Simple equality - MySQL",
			dialect:  "mysql",
			query:    map[string]interface{}{"name": "John"},
			expected: "`name` = ?",
			args:     []interface{}{"John"},
		},
		{
			name:     "Simple equality - PostgreSQL",
			dialect:  "postgres",
			query:    map[string]interface{}{"name": "John"},
			expected: `"name" = $1`,
			args:     []interface{}{"John"},
		},
		{
			name:     "Simple equality - SQLite",
			dialect:  "sqlite",
			query:    map[string]interface{}{"name": "John"},
			expected: `"name" = ?`,
			args:     []interface{}{"John"},
		},
		{
			name:    "Multiple fields",
			dialect: "mysql",
			query: map[string]interface{}{
				"name": "John",
				"age":  25,
			},
			expected: "`name` = ? AND `age` = ?",
			args:     []interface{}{"John", 25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewQueryTranslator(tt.dialect)
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_ComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name: "Greater than",
			query: map[string]interface{}{
				"age": map[string]interface{}{"$gt": 18},
			},
			expected: "(`age` > ?)",
			args:     []interface{}{18},
		},
		{
			name: "Greater than or equal",
			query: map[string]interface{}{
				"age": map[string]interface{}{"$gte": 18},
			},
			expected: "(`age` >= ?)",
			args:     []interface{}{18},
		},
		{
			name: "Less than",
			query: map[string]interface{}{
				"age": map[string]interface{}{"$lt": 65},
			},
			expected: "(`age` < ?)",
			args:     []interface{}{65},
		},
		{
			name: "Less than or equal",
			query: map[string]interface{}{
				"age": map[string]interface{}{"$lte": 65},
			},
			expected: "(`age` <= ?)",
			args:     []interface{}{65},
		},
		{
			name: "Not equal",
			query: map[string]interface{}{
				"status": map[string]interface{}{"$ne": "inactive"},
			},
			expected: "(`status` != ?)",
			args:     []interface{}{"inactive"},
		},
		{
			name: "Multiple operators on same field",
			query: map[string]interface{}{
				"age": map[string]interface{}{
					"$gte": 18,
					"$lt":  65,
				},
			},
			expected: "(`age` >= ? AND `age` < ?)",
			args:     []interface{}{18, 65},
		},
	}

	translator := NewQueryTranslator("mysql")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_InOperator(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name: "$in with multiple values",
			query: map[string]interface{}{
				"status": map[string]interface{}{
					"$in": []interface{}{"active", "pending", "approved"},
				},
			},
			expected: "(`status` IN (?, ?, ?))",
			args:     []interface{}{"active", "pending", "approved"},
		},
		{
			name: "$in with single value",
			query: map[string]interface{}{
				"id": map[string]interface{}{
					"$in": []interface{}{123},
				},
			},
			expected: "(`id` IN (?))",
			args:     []interface{}{123},
		},
		{
			name: "$in with empty array",
			query: map[string]interface{}{
				"status": map[string]interface{}{
					"$in": []interface{}{},
				},
			},
			expected: "(1=0)",
			args:     []interface{}{},
		},
		{
			name: "$nin with multiple values",
			query: map[string]interface{}{
				"status": map[string]interface{}{
					"$nin": []interface{}{"deleted", "banned"},
				},
			},
			expected: "(`status` NOT IN (?, ?))",
			args:     []interface{}{"deleted", "banned"},
		},
	}

	translator := NewQueryTranslator("mysql")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_LogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name: "$or with simple conditions",
			query: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"status": "active"},
					map[string]interface{}{"priority": "high"},
				},
			},
			expected: "((`status` = ?) OR (`priority` = ?))",
			args:     []interface{}{"active", "high"},
		},
		{
			name: "$and with simple conditions",
			query: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"age": map[string]interface{}{"$gte": 18}},
					map[string]interface{}{"status": "active"},
				},
			},
			expected: "(`age` >= ?) AND (`status` = ?)",
			args:     []interface{}{18, "active"},
		},
		{
			name: "Complex query with $or and regular fields",
			query: map[string]interface{}{
				"department": "engineering",
				"$or": []interface{}{
					map[string]interface{}{"role": "senior"},
					map[string]interface{}{"experience": map[string]interface{}{"$gte": 5}},
				},
			},
			expected: "`department` = ? AND ((`role` = ?) OR ((`experience` >= ?)))",
			args:     []interface{}{"engineering", "senior", 5},
		},
		{
			name: "$not operator",
			query: map[string]interface{}{
				"$not": map[string]interface{}{
					"status": "deleted",
				},
			},
			expected: "NOT (`status` = ?)",
			args:     []interface{}{"deleted"},
		},
	}

	translator := NewQueryTranslator("mysql")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_ExistsOperator(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name: "$exists true",
			query: map[string]interface{}{
				"email": map[string]interface{}{
					"$exists": true,
				},
			},
			expected: "(`email` IS NOT NULL)",
			args:     []interface{}{},
		},
		{
			name: "$exists false",
			query: map[string]interface{}{
				"deletedAt": map[string]interface{}{
					"$exists": false,
				},
			},
			expected: "(`deletedAt` IS NULL)",
			args:     []interface{}{},
		},
	}

	translator := NewQueryTranslator("mysql")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_RegexOperator(t *testing.T) {
	tests := []struct {
		name     string
		dialect  string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name:    "Regex in MySQL",
			dialect: "mysql",
			query: map[string]interface{}{
				"name": map[string]interface{}{
					"$regex": "^John.*",
				},
			},
			expected: "(`name` LIKE ?)",
			args:     []interface{}{"John%"},
		},
		{
			name:    "Regex in PostgreSQL",
			dialect: "postgres",
			query: map[string]interface{}{
				"name": map[string]interface{}{
					"$regex": "^John.*",
				},
			},
			expected: "(`name` ~ $1)",
			args:     []interface{}{"^John.*"},
		},
		{
			name:    "Regex in SQLite",
			dialect: "sqlite",
			query: map[string]interface{}{
				"email": map[string]interface{}{
					"$regex": ".*@example.com$",
				},
			},
			expected: "(`email` LIKE ?)",
			args:     []interface{}{"%@example.com"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewQueryTranslator(tt.dialect)
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_JSONFields(t *testing.T) {
	tests := []struct {
		name     string
		dialect  string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name:    "JSON field access - MySQL",
			dialect: "mysql",
			query: map[string]interface{}{
				"profile.age": 25,
			},
			expected: "JSON_UNQUOTE(JSON_EXTRACT(`profile`, '$.age')) = ?",
			args:     []interface{}{25},
		},
		{
			name:    "JSON field access - PostgreSQL",
			dialect: "postgres",
			query: map[string]interface{}{
				"address.city": "New York",
			},
			expected: `"address"->>'city' = $1`,
			args:     []interface{}{"New York"},
		},
		{
			name:    "JSON field access - SQLite",
			dialect: "sqlite",
			query: map[string]interface{}{
				"metadata.version": "1.0",
			},
			expected: `json_extract("metadata", '$.version') = ?`,
			args:     []interface{}{"1.0"},
		},
		{
			name:    "Nested JSON field - PostgreSQL",
			dialect: "postgres",
			query: map[string]interface{}{
				"user.profile.settings.theme": "dark",
			},
			expected: `"user"->'profile'->'settings'->>'theme' = $1`,
			args:     []interface{}{"dark"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewQueryTranslator(tt.dialect)
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_TranslateSort(t *testing.T) {
	tests := []struct {
		name     string
		sort     map[string]interface{}
		expected string
	}{
		{
			name: "Single field ascending",
			sort: map[string]interface{}{
				"name": 1,
			},
			expected: "`name` ASC",
		},
		{
			name: "Single field descending",
			sort: map[string]interface{}{
				"createdAt": -1,
			},
			expected: "`createdAt` DESC",
		},
		{
			name: "Multiple fields",
			sort: map[string]interface{}{
				"priority": -1,
				"name":     1,
			},
			expected: "`priority` DESC, `name` ASC",
		},
	}

	translator := NewQueryTranslator("mysql")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderBy, err := translator.TranslateSort(tt.sort)
			
			if err != nil {
				t.Errorf("TranslateSort() error = %v", err)
				return
			}
			
			// Note: map iteration order is not guaranteed, so we need to check both possibilities
			if orderBy != tt.expected && len(tt.sort) > 1 {
				// For multiple fields, check if it's the reverse order
				expectedReverse := "`name` ASC, `priority` DESC"
				if orderBy != expectedReverse {
					t.Errorf("TranslateSort() orderBy = %v, want %v or %v", orderBy, tt.expected, expectedReverse)
				}
			} else if orderBy != tt.expected {
				t.Errorf("TranslateSort() orderBy = %v, want %v", orderBy, tt.expected)
			}
		})
	}
}

func TestQueryTranslator_BuildSelectQuery(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		query    map[string]interface{}
		options  map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name:     "Simple select all",
			table:    "users",
			query:    map[string]interface{}{},
			options:  map[string]interface{}{},
			expected: "SELECT * FROM `users`",
			args:     []interface{}{},
		},
		{
			name:  "Select with where clause",
			table: "users",
			query: map[string]interface{}{
				"status": "active",
			},
			options:  map[string]interface{}{},
			expected: "SELECT * FROM `users` WHERE `status` = ?",
			args:     []interface{}{"active"},
		},
		{
			name:  "Select with where, order, and limit",
			table: "users",
			query: map[string]interface{}{
				"age": map[string]interface{}{"$gte": 18},
			},
			options: map[string]interface{}{
				"$sort":  map[string]interface{}{"name": 1},
				"$limit": 10,
			},
			expected: "SELECT * FROM `users` WHERE (`age` >= ?) ORDER BY `name` ASC LIMIT 10",
			args:     []interface{}{18},
		},
		{
			name:  "Select with projection",
			table: "users",
			query: map[string]interface{}{},
			options: map[string]interface{}{
				"$fields": map[string]interface{}{
					"name":  1,
					"email": 1,
				},
			},
			expected: "SELECT `id`, `name`, `email` FROM `users`",
			args:     []interface{}{},
		},
		{
			name:  "Complex query",
			table: "orders",
			query: map[string]interface{}{
				"status": "pending",
				"$or": []interface{}{
					map[string]interface{}{"priority": "high"},
					map[string]interface{}{"amount": map[string]interface{}{"$gte": 1000}},
				},
			},
			options: map[string]interface{}{
				"$sort":  map[string]interface{}{"createdAt": -1},
				"$limit": 50,
				"$skip":  10,
			},
			expected: "SELECT * FROM `orders` WHERE `status` = ? AND ((`priority` = ?) OR ((`amount` >= ?))) ORDER BY `createdAt` DESC LIMIT 50 OFFSET 10",
			args:     []interface{}{"pending", "high", 1000},
		},
	}

	translator := NewQueryTranslator("mysql")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := translator.BuildSelectQuery(tt.table, tt.query, tt.options)
			
			if err != nil {
				t.Errorf("BuildSelectQuery() error = %v", err)
				return
			}
			
			if query != tt.expected {
				t.Errorf("BuildSelectQuery() query = %v, want %v", query, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("BuildSelectQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func TestQueryTranslator_ArrayOperators(t *testing.T) {
	tests := []struct {
		name     string
		dialect  string
		query    map[string]interface{}
		expected string
		args     []interface{}
	}{
		{
			name:    "$size operator - MySQL",
			dialect: "mysql",
			query: map[string]interface{}{
				"tags": map[string]interface{}{
					"$size": 3,
				},
			},
			expected: "(JSON_LENGTH(`tags`) = ?)",
			args:     []interface{}{3},
		},
		{
			name:    "$size operator - PostgreSQL",
			dialect: "postgres",
			query: map[string]interface{}{
				"items": map[string]interface{}{
					"$size": 5,
				},
			},
			expected: "(jsonb_array_length(`items`) = $1)",
			args:     []interface{}{5},
		},
		{
			name:    "$size operator - SQLite",
			dialect: "sqlite",
			query: map[string]interface{}{
				"categories": map[string]interface{}{
					"$size": 2,
				},
			},
			expected: "(json_array_length(`categories`) = ?)",
			args:     []interface{}{2},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewQueryTranslator(tt.dialect)
			whereClause, args, err := translator.TranslateQuery(tt.query)
			
			if err != nil {
				t.Errorf("TranslateQuery() error = %v", err)
				return
			}
			
			if whereClause != tt.expected {
				t.Errorf("TranslateQuery() whereClause = %v, want %v", whereClause, tt.expected)
			}
			
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("TranslateQuery() args = %v, want %v", args, tt.args)
			}
		})
	}
}

func BenchmarkQueryTranslator_SimpleQuery(b *testing.B) {
	translator := NewQueryTranslator("mysql")
	query := map[string]interface{}{
		"name":   "John",
		"age":    25,
		"status": "active",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = translator.TranslateQuery(query)
	}
}

func BenchmarkQueryTranslator_ComplexQuery(b *testing.B) {
	translator := NewQueryTranslator("mysql")
	query := map[string]interface{}{
		"department": "engineering",
		"age": map[string]interface{}{
			"$gte": 18,
			"$lt":  65,
		},
		"$or": []interface{}{
			map[string]interface{}{"role": "senior"},
			map[string]interface{}{"experience": map[string]interface{}{"$gte": 5}},
			map[string]interface{}{"skills": map[string]interface{}{"$in": []interface{}{"go", "python", "javascript"}}},
		},
		"profile.active": true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = translator.TranslateQuery(query)
	}
}