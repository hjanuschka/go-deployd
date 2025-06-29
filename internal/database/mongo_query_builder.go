package database

import (
	"encoding/json"
	"fmt"
)

// MongoQueryBuilder builds MongoDB-style queries that can be translated to SQL
type MongoQueryBuilder struct {
	conditions map[string]interface{}
}

// NewMongoQueryBuilder creates a new MongoDB-style query builder
func NewMongoQueryBuilder() *MongoQueryBuilder {
	return &MongoQueryBuilder{
		conditions: make(map[string]interface{}),
	}
}

// ParseMongoQuery parses a MongoDB query from JSON string or map
func ParseMongoQuery(query interface{}) (map[string]interface{}, error) {
	switch q := query.(type) {
	case string:
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(q), &result); err != nil {
			return nil, fmt.Errorf("invalid JSON query: %w", err)
		}
		return result, nil
	case map[string]interface{}:
		return q, nil
	case nil:
		return make(map[string]interface{}), nil
	default:
		return nil, fmt.Errorf("query must be a JSON string or map, got %T", query)
	}
}

// Where adds a simple equality condition
func (mqb *MongoQueryBuilder) Where(field string, value interface{}) *MongoQueryBuilder {
	mqb.conditions[field] = value
	return mqb
}

// WhereOperator adds a condition with a MongoDB operator
func (mqb *MongoQueryBuilder) WhereOperator(field, operator string, value interface{}) *MongoQueryBuilder {
	if existing, exists := mqb.conditions[field]; exists {
		if existingMap, ok := existing.(map[string]interface{}); ok {
			existingMap[operator] = value
		} else {
			// Convert existing value to $eq operator
			mqb.conditions[field] = map[string]interface{}{
				"$eq":    existing,
				operator: value,
			}
		}
	} else {
		mqb.conditions[field] = map[string]interface{}{
			operator: value,
		}
	}
	return mqb
}

// Equals adds an equality condition ($eq)
func (mqb *MongoQueryBuilder) Equals(field string, value interface{}) *MongoQueryBuilder {
	return mqb.Where(field, value)
}

// NotEquals adds a not-equals condition ($ne)
func (mqb *MongoQueryBuilder) NotEquals(field string, value interface{}) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$ne", value)
}

// GreaterThan adds a greater-than condition ($gt)
func (mqb *MongoQueryBuilder) GreaterThan(field string, value interface{}) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$gt", value)
}

// GreaterThanOrEqual adds a greater-than-or-equal condition ($gte)
func (mqb *MongoQueryBuilder) GreaterThanOrEqual(field string, value interface{}) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$gte", value)
}

// LessThan adds a less-than condition ($lt)
func (mqb *MongoQueryBuilder) LessThan(field string, value interface{}) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$lt", value)
}

// LessThanOrEqual adds a less-than-or-equal condition ($lte)
func (mqb *MongoQueryBuilder) LessThanOrEqual(field string, value interface{}) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$lte", value)
}

// In adds an in condition ($in)
func (mqb *MongoQueryBuilder) In(field string, values []interface{}) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$in", values)
}

// NotIn adds a not-in condition ($nin)
func (mqb *MongoQueryBuilder) NotIn(field string, values []interface{}) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$nin", values)
}

// Exists adds an exists condition ($exists)
func (mqb *MongoQueryBuilder) Exists(field string, exists bool) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$exists", exists)
}

// Regex adds a regex condition ($regex)
func (mqb *MongoQueryBuilder) Regex(field, pattern string) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$regex", pattern)
}

// Size adds an array size condition ($size)
func (mqb *MongoQueryBuilder) Size(field string, size int) *MongoQueryBuilder {
	return mqb.WhereOperator(field, "$size", size)
}

// Between adds a range condition (combination of $gte and $lte)
func (mqb *MongoQueryBuilder) Between(field string, min, max interface{}) *MongoQueryBuilder {
	mqb.conditions[field] = map[string]interface{}{
		"$gte": min,
		"$lte": max,
	}
	return mqb
}

// Or adds an OR condition
func (mqb *MongoQueryBuilder) Or(conditions ...map[string]interface{}) *MongoQueryBuilder {
	if len(conditions) == 0 {
		return mqb
	}
	
	// Convert MongoQueryBuilder conditions to map
	orConditions := make([]interface{}, 0, len(conditions))
	for _, cond := range conditions {
		orConditions = append(orConditions, cond)
	}
	
	if existing, exists := mqb.conditions["$or"]; exists {
		if existingOr, ok := existing.([]interface{}); ok {
			mqb.conditions["$or"] = append(existingOr, orConditions...)
		}
	} else {
		mqb.conditions["$or"] = orConditions
	}
	
	return mqb
}

// And adds an AND condition
func (mqb *MongoQueryBuilder) And(conditions ...map[string]interface{}) *MongoQueryBuilder {
	if len(conditions) == 0 {
		return mqb
	}
	
	// Convert MongoQueryBuilder conditions to map
	andConditions := make([]interface{}, 0, len(conditions))
	for _, cond := range conditions {
		andConditions = append(andConditions, cond)
	}
	
	if existing, exists := mqb.conditions["$and"]; exists {
		if existingAnd, ok := existing.([]interface{}); ok {
			mqb.conditions["$and"] = append(existingAnd, andConditions...)
		}
	} else {
		mqb.conditions["$and"] = andConditions
	}
	
	return mqb
}

// Not adds a NOT condition
func (mqb *MongoQueryBuilder) Not(condition map[string]interface{}) *MongoQueryBuilder {
	mqb.conditions["$not"] = condition
	return mqb
}

// ToMap returns the query as a map for MongoDB compatibility
func (mqb *MongoQueryBuilder) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range mqb.conditions {
		result[k] = v
	}
	return result
}

// ToJSON returns the query as a JSON string
func (mqb *MongoQueryBuilder) ToJSON() (string, error) {
	queryMap := mqb.ToMap()
	jsonBytes, err := json.Marshal(queryMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal query to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// Clone creates a copy of the query builder
func (mqb *MongoQueryBuilder) Clone() *MongoQueryBuilder {
	clone := NewMongoQueryBuilder()
	for k, v := range mqb.conditions {
		clone.conditions[k] = v
	}
	return clone
}

// IsEmpty checks if the query has any conditions
func (mqb *MongoQueryBuilder) IsEmpty() bool {
	return len(mqb.conditions) == 0
}

// Clear removes all conditions
func (mqb *MongoQueryBuilder) Clear() *MongoQueryBuilder {
	mqb.conditions = make(map[string]interface{})
	return mqb
}

// MongoQueryOptions represents MongoDB-style query options
type MongoQueryOptions struct {
	Sort   map[string]interface{} `json:"$sort,omitempty"`
	Limit  *int                   `json:"$limit,omitempty"`
	Skip   *int                   `json:"$skip,omitempty"`
	Fields map[string]interface{} `json:"$fields,omitempty"`
}

// NewMongoQueryOptions creates new query options
func NewMongoQueryOptions() *MongoQueryOptions {
	return &MongoQueryOptions{}
}

// SetSort sets the sort options
func (opts *MongoQueryOptions) SetSort(field string, direction int) *MongoQueryOptions {
	if opts.Sort == nil {
		opts.Sort = make(map[string]interface{})
	}
	opts.Sort[field] = direction
	return opts
}

// SetLimit sets the limit
func (opts *MongoQueryOptions) SetLimit(limit int) *MongoQueryOptions {
	opts.Limit = &limit
	return opts
}

// SetSkip sets the skip/offset
func (opts *MongoQueryOptions) SetSkip(skip int) *MongoQueryOptions {
	opts.Skip = &skip
	return opts
}

// SetFields sets the field projection
func (opts *MongoQueryOptions) SetFields(fields map[string]interface{}) *MongoQueryOptions {
	opts.Fields = fields
	return opts
}

// IncludeFields sets fields to include (projection)
func (opts *MongoQueryOptions) IncludeFields(fields ...string) *MongoQueryOptions {
	if opts.Fields == nil {
		opts.Fields = make(map[string]interface{})
	}
	for _, field := range fields {
		opts.Fields[field] = 1
	}
	return opts
}

// ExcludeFields sets fields to exclude (projection)
func (opts *MongoQueryOptions) ExcludeFields(fields ...string) *MongoQueryOptions {
	if opts.Fields == nil {
		opts.Fields = make(map[string]interface{})
	}
	for _, field := range fields {
		opts.Fields[field] = 0
	}
	return opts
}

// ToMap converts options to map for API compatibility
func (opts *MongoQueryOptions) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	
	if opts.Sort != nil {
		result["$sort"] = opts.Sort
	}
	if opts.Limit != nil {
		result["$limit"] = *opts.Limit
	}
	if opts.Skip != nil {
		result["$skip"] = *opts.Skip
	}
	if opts.Fields != nil {
		result["$fields"] = opts.Fields
	}
	
	return result
}

// Convenience functions for common query patterns

// FindActive creates a query for active records
func FindActive() *MongoQueryBuilder {
	return NewMongoQueryBuilder().Where("status", "active")
}

// FindByID creates a query for a specific ID
func FindByID(id interface{}) *MongoQueryBuilder {
	return NewMongoQueryBuilder().Where("id", id)
}

// FindByIDs creates a query for multiple IDs
func FindByIDs(ids []interface{}) *MongoQueryBuilder {
	return NewMongoQueryBuilder().In("id", ids)
}

// FindRecent creates a query for recent records
func FindRecent(field string, limit int) *MongoQueryBuilder {
	// This would typically be used with options to sort by creation date
	return NewMongoQueryBuilder().Exists(field, true)
}

// FindByDateRange creates a query for records within a date range
func FindByDateRange(field string, start, end interface{}) *MongoQueryBuilder {
	return NewMongoQueryBuilder().Between(field, start, end)
}

// FindByUser creates a query for records belonging to a user
func FindByUser(userID interface{}) *MongoQueryBuilder {
	return NewMongoQueryBuilder().Where("userId", userID)
}

// FindPublic creates a query for public records
func FindPublic() *MongoQueryBuilder {
	return NewMongoQueryBuilder().Where("public", true)
}

// Example usage:
//
// // Build a complex query
// query := NewMongoQueryBuilder().
//     Where("department", "engineering").
//     GreaterThanOrEqual("age", 18).
//     In("skills", []interface{}{"go", "python"}).
//     Or(
//         map[string]interface{}{"role": "senior"},
//         map[string]interface{}{"experience": map[string]interface{}{"$gte": 5}},
//     )
//
// // Convert to map for API
// queryMap := query.ToMap()
//
// // Use with query translator
// translator := NewQueryTranslator("mysql")
// sqlWhere, args, err := translator.TranslateQuery(queryMap)
//
// // Or use with raw query methods
// results, err := store.FindWithRawQuery(ctx, queryMap, options)