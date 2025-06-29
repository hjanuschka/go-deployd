package database

import (
	"fmt"
	"strings"
)

// BaseQueryBuilder provides a basic implementation of QueryBuilder
type BaseQueryBuilder struct {
	conditions []QueryCondition
	orGroups   [][]QueryCondition
}

type QueryCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() QueryBuilder {
	return &BaseQueryBuilder{
		conditions: make([]QueryCondition, 0),
		orGroups:   make([][]QueryCondition, 0),
	}
}

func (q *BaseQueryBuilder) Where(field string, operator string, value interface{}) QueryBuilder {
	q.conditions = append(q.conditions, QueryCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return q
}

func (q *BaseQueryBuilder) WhereIn(field string, values []interface{}) QueryBuilder {
	return q.Where(field, "$in", values)
}

func (q *BaseQueryBuilder) WhereNotIn(field string, values []interface{}) QueryBuilder {
	return q.Where(field, "$nin", values)
}

func (q *BaseQueryBuilder) WhereNull(field string) QueryBuilder {
	return q.Where(field, "$eq", nil)
}

func (q *BaseQueryBuilder) WhereNotNull(field string) QueryBuilder {
	return q.Where(field, "$ne", nil)
}

func (q *BaseQueryBuilder) WhereRegex(field string, pattern string) QueryBuilder {
	return q.Where(field, "$regex", pattern)
}

func (q *BaseQueryBuilder) Or(conditions ...QueryBuilder) QueryBuilder {
	orGroup := make([]QueryCondition, 0)
	for _, cond := range conditions {
		if baseBuilder, ok := cond.(*BaseQueryBuilder); ok {
			orGroup = append(orGroup, baseBuilder.conditions...)
		}
	}
	q.orGroups = append(q.orGroups, orGroup)
	return q
}

func (q *BaseQueryBuilder) And(conditions ...QueryBuilder) QueryBuilder {
	for _, cond := range conditions {
		if baseBuilder, ok := cond.(*BaseQueryBuilder); ok {
			q.conditions = append(q.conditions, baseBuilder.conditions...)
		}
	}
	return q
}

func (q *BaseQueryBuilder) Clone() QueryBuilder {
	clone := &BaseQueryBuilder{
		conditions: make([]QueryCondition, len(q.conditions)),
		orGroups:   make([][]QueryCondition, len(q.orGroups)),
	}
	copy(clone.conditions, q.conditions)
	copy(clone.orGroups, q.orGroups)
	return clone
}

func (q *BaseQueryBuilder) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Add regular conditions
	for _, cond := range q.conditions {
		if cond.Operator == "$eq" || cond.Operator == "=" {
			result[cond.Field] = cond.Value
		} else {
			if existing, exists := result[cond.Field]; exists {
				if existingMap, ok := existing.(map[string]interface{}); ok {
					existingMap[cond.Operator] = cond.Value
				} else {
					result[cond.Field] = map[string]interface{}{
						cond.Operator: cond.Value,
					}
				}
			} else {
				result[cond.Field] = map[string]interface{}{
					cond.Operator: cond.Value,
				}
			}
		}
	}

	// Add OR groups
	if len(q.orGroups) > 0 {
		orConditions := make([]map[string]interface{}, 0)
		for _, group := range q.orGroups {
			groupMap := make(map[string]interface{})
			for _, cond := range group {
				if cond.Operator == "$eq" || cond.Operator == "=" {
					groupMap[cond.Field] = cond.Value
				} else {
					groupMap[cond.Field] = map[string]interface{}{
						cond.Operator: cond.Value,
					}
				}
			}
			orConditions = append(orConditions, groupMap)
		}
		result["$or"] = orConditions
	}

	return result
}

// SQLQueryBuilder extends BaseQueryBuilder for SQL-specific functionality
type SQLQueryBuilder struct {
	*BaseQueryBuilder
	rawConditions []RawCondition
	// columnChecker determines if a field should use direct column access
	// If nil, defaults to JSON access for all fields
	columnChecker func(field string) bool
}

type RawCondition struct {
	SQL  string
	Args []interface{}
}

func NewSQLQueryBuilder() *SQLQueryBuilder {
	return &SQLQueryBuilder{
		BaseQueryBuilder: &BaseQueryBuilder{
			conditions: make([]QueryCondition, 0),
			orGroups:   make([][]QueryCondition, 0),
		},
		rawConditions: make([]RawCondition, 0),
		columnChecker: nil, // Default to JSON access
	}
}

// SetColumnChecker sets a function to determine if a field should use direct column access
func (q *SQLQueryBuilder) SetColumnChecker(checker func(field string) bool) *SQLQueryBuilder {
	q.columnChecker = checker
	return q
}

// WhereRaw adds a raw SQL condition
func (q *SQLQueryBuilder) WhereRaw(sql string, args ...interface{}) *SQLQueryBuilder {
	q.rawConditions = append(q.rawConditions, RawCondition{
		SQL:  sql,
		Args: args,
	})
	return q
}

// ToSQL converts the query to SQL WHERE clause
func (q *SQLQueryBuilder) ToSQL() (string, []interface{}) {
	if len(q.conditions) == 0 && len(q.orGroups) == 0 && len(q.rawConditions) == 0 {
		return "", nil
	}

	var whereParts []string
	var args []interface{}

	// Add regular conditions
	for _, cond := range q.conditions {
		sqlOperator, argCount := q.convertOperator(cond.Operator)
		fieldRef := q.getFieldReference(cond.Field)
		
		if argCount == 0 {
			whereParts = append(whereParts, fmt.Sprintf("%s %s", fieldRef, sqlOperator))
		} else if argCount == 1 {
			whereParts = append(whereParts, fmt.Sprintf("%s %s ?", fieldRef, sqlOperator))
			// Special handling for regex operators
			if cond.Operator == "$regex" {
				args = append(args, q.regexToLike(cond.Value))
			} else {
				args = append(args, cond.Value)
			}
		} else {
			// Handle IN/NOT IN operators
			if values, ok := cond.Value.([]interface{}); ok {
				placeholders := strings.Repeat("?,", len(values))
				placeholders = placeholders[:len(placeholders)-1] // remove trailing comma
				whereParts = append(whereParts, fmt.Sprintf("%s %s (%s)", fieldRef, sqlOperator, placeholders))
				args = append(args, values...)
			}
		}
	}

	// Add OR groups
	for _, group := range q.orGroups {
		if len(group) > 0 {
			var orParts []string
			for _, cond := range group {
				sqlOperator, argCount := q.convertOperator(cond.Operator)
				fieldRef := q.getFieldReference(cond.Field)
				
				if argCount == 0 {
					orParts = append(orParts, fmt.Sprintf("%s %s", fieldRef, sqlOperator))
				} else if argCount == 1 {
					orParts = append(orParts, fmt.Sprintf("%s %s ?", fieldRef, sqlOperator))
					// Special handling for regex operators
					if cond.Operator == "$regex" {
						args = append(args, q.regexToLike(cond.Value))
					} else {
						args = append(args, cond.Value)
					}
				} else {
					// Handle IN/NOT IN operators
					if values, ok := cond.Value.([]interface{}); ok {
						placeholders := strings.Repeat("?,", len(values))
						placeholders = placeholders[:len(placeholders)-1]
						orParts = append(orParts, fmt.Sprintf("%s %s (%s)", fieldRef, sqlOperator, placeholders))
						args = append(args, values...)
					}
				}
			}
			if len(orParts) > 0 {
				whereParts = append(whereParts, fmt.Sprintf("(%s)", strings.Join(orParts, " OR ")))
			}
		}
	}

	// Add raw conditions
	for _, rawCond := range q.rawConditions {
		whereParts = append(whereParts, rawCond.SQL)
		args = append(args, rawCond.Args...)
	}

	if len(whereParts) == 0 {
		return "", nil
	}

	sql := strings.Join(whereParts, " AND ")
	
	// Debug logging - print the actual SQL being generated
	fmt.Printf("DEBUG: SQLQueryBuilder generated SQL: %s\n", sql)
	fmt.Printf("DEBUG: SQLQueryBuilder args: %v\n", args)
	
	return sql, args
}

// getFieldReference returns the appropriate field reference (column or JSON extraction)
func (q *SQLQueryBuilder) getFieldReference(field string) string {
	// If we have a column checker and the field has a column, use direct access
	if q.columnChecker != nil && q.columnChecker(field) {
		// For direct column access, quote the field name to handle special characters
		fieldRef := fmt.Sprintf("\"%s\"", field)
		fmt.Printf("DEBUG: Field '%s' has column, using direct access: %s\n", field, fieldRef)
		return fieldRef
	}
	
	// Default to JSON extraction for backward compatibility
	fieldRef := fmt.Sprintf("JSON_EXTRACT(data, '$.%s')", field)
	fmt.Printf("DEBUG: Field '%s' no column, using JSON extraction: %s\n", field, fieldRef)
	return fieldRef
}

// convertOperator converts MongoDB operators to SQL operators
func (q *SQLQueryBuilder) convertOperator(mongoOp string) (string, int) {
	switch mongoOp {
	case "$eq", "=":
		return "=", 1
	case "$ne", "!=":
		return "!=", 1
	case "$gt", ">":
		return ">", 1
	case "$gte", ">=":
		return ">=", 1
	case "$lt", "<":
		return "<", 1
	case "$lte", "<=":
		return "<=", 1
	case "$in":
		return "IN", 2 // Special case for multiple values
	case "$nin":
		return "NOT IN", 2 // Special case for multiple values
	case "$regex":
		return "LIKE", 1 // Convert regex to LIKE for basic pattern matching
	case "$exists":
		return "IS NOT NULL", 0
	default:
		return "=", 1 // Default to equality
	}
}

// regexToLike converts regex patterns to SQL LIKE patterns
func (q *SQLQueryBuilder) regexToLike(value interface{}) string {
	pattern, ok := value.(string)
	if !ok {
		return fmt.Sprintf("%v", value)
	}

	// Simple conversion for basic regex patterns
	// This handles the most common cases used in search functionality
	
	// Handle anchors first
	isStartAnchored := strings.HasPrefix(pattern, "^")
	isEndAnchored := strings.HasSuffix(pattern, "$")
	
	// Remove anchors for processing
	if isStartAnchored {
		pattern = strings.TrimPrefix(pattern, "^")
	}
	if isEndAnchored {
		pattern = strings.TrimSuffix(pattern, "$")
	}
	
	// Escape SQL LIKE special characters
	pattern = strings.ReplaceAll(pattern, "%", "\\%")
	pattern = strings.ReplaceAll(pattern, "_", "\\_")
	
	// Convert basic regex patterns to LIKE patterns
	// Handle .* (any characters) -> %
	pattern = strings.ReplaceAll(pattern, ".*", "%")
	// Handle .+ (one or more characters) -> _%
	pattern = strings.ReplaceAll(pattern, ".+", "_%")
	// Handle . (single character) -> _
	pattern = strings.ReplaceAll(pattern, ".", "_")
	
	// Apply anchoring
	if !isStartAnchored {
		pattern = "%" + pattern
	}
	if !isEndAnchored {
		pattern = pattern + "%"
	}
	
	return pattern
}

// BaseUpdateBuilder provides a basic implementation of UpdateBuilder
type BaseUpdateBuilder struct {
	operations map[string]map[string]interface{}
}

func NewUpdateBuilder() UpdateBuilder {
	return &BaseUpdateBuilder{
		operations: make(map[string]map[string]interface{}),
	}
}

func (u *BaseUpdateBuilder) Set(field string, value interface{}) UpdateBuilder {
	if u.operations["$set"] == nil {
		u.operations["$set"] = make(map[string]interface{})
	}
	u.operations["$set"][field] = value
	return u
}

func (u *BaseUpdateBuilder) Unset(field string) UpdateBuilder {
	if u.operations["$unset"] == nil {
		u.operations["$unset"] = make(map[string]interface{})
	}
	u.operations["$unset"][field] = ""
	return u
}

func (u *BaseUpdateBuilder) Inc(field string, value interface{}) UpdateBuilder {
	if u.operations["$inc"] == nil {
		u.operations["$inc"] = make(map[string]interface{})
	}
	u.operations["$inc"][field] = value
	return u
}

func (u *BaseUpdateBuilder) Push(field string, value interface{}) UpdateBuilder {
	if u.operations["$push"] == nil {
		u.operations["$push"] = make(map[string]interface{})
	}
	u.operations["$push"][field] = value
	return u
}

func (u *BaseUpdateBuilder) Pull(field string, value interface{}) UpdateBuilder {
	if u.operations["$pull"] == nil {
		u.operations["$pull"] = make(map[string]interface{})
	}
	u.operations["$pull"][field] = value
	return u
}

func (u *BaseUpdateBuilder) AddToSet(field string, value interface{}) UpdateBuilder {
	if u.operations["$addToSet"] == nil {
		u.operations["$addToSet"] = make(map[string]interface{})
	}
	u.operations["$addToSet"][field] = value
	return u
}

func (u *BaseUpdateBuilder) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for op, fields := range u.operations {
		result[op] = fields
	}
	return result
}
