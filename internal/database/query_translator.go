package database

import (
	"fmt"
	"strings"
)

// QueryTranslator converts MongoDB-style queries to SQL
type QueryTranslator struct {
	dialect string // "sqlite", "mysql", "postgres"
}

// NewQueryTranslator creates a new query translator for the specified dialect
func NewQueryTranslator(dialect string) *QueryTranslator {
	return &QueryTranslator{dialect: dialect}
}

// TranslateQuery converts a MongoDB query to SQL WHERE clause
func (qt *QueryTranslator) TranslateQuery(mongoQuery interface{}) (string, []interface{}, error) {
	if mongoQuery == nil {
		return "", nil, nil
	}

	query, ok := mongoQuery.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("query must be an object")
	}

	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	for field, value := range query {
		condition, newArgs, newArgIndex, err := qt.translateCondition(field, value, argIndex)
		if err != nil {
			return "", nil, err
		}
		
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, newArgs...)
			argIndex = newArgIndex
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}

	return strings.Join(conditions, " AND "), args, nil
}

// translateCondition converts a single MongoDB condition to SQL
func (qt *QueryTranslator) translateCondition(field string, value interface{}, argIndex int) (string, []interface{}, int, error) {
	// Handle special MongoDB operators
	if strings.HasPrefix(field, "$") {
		return qt.translateOperator(field, value, argIndex)
	}

	// Handle nested operators
	if valueMap, ok := value.(map[string]interface{}); ok {
		return qt.translateFieldOperators(field, valueMap, argIndex)
	}

	// Simple equality
	placeholder := qt.getPlaceholder(argIndex)
	return fmt.Sprintf("%s = %s", qt.escapeField(field), placeholder), []interface{}{value}, argIndex + 1, nil
}

// translateOperator handles top-level MongoDB operators like $or, $and
func (qt *QueryTranslator) translateOperator(operator string, value interface{}, argIndex int) (string, []interface{}, int, error) {
	switch operator {
	case "$or":
		conditions, ok := value.([]interface{})
		if !ok {
			return "", nil, argIndex, fmt.Errorf("$or must be an array")
		}
		
		orClauses := []string{}
		args := []interface{}{}
		
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}
			
			clause, condArgs, newArgIndex, err := qt.translateQuery(condMap, argIndex)
			if err != nil {
				return "", nil, argIndex, err
			}
			
			if clause != "" {
				orClauses = append(orClauses, "("+clause+")")
				args = append(args, condArgs...)
				argIndex = newArgIndex
			}
		}
		
		if len(orClauses) > 0 {
			return "(" + strings.Join(orClauses, " OR ") + ")", args, argIndex, nil
		}
		
	case "$and":
		conditions, ok := value.([]interface{})
		if !ok {
			return "", nil, argIndex, fmt.Errorf("$and must be an array")
		}
		
		andClauses := []string{}
		args := []interface{}{}
		
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}
			
			clause, condArgs, newArgIndex, err := qt.translateQuery(condMap, argIndex)
			if err != nil {
				return "", nil, argIndex, err
			}
			
			if clause != "" {
				andClauses = append(andClauses, clause)
				args = append(args, condArgs...)
				argIndex = newArgIndex
			}
		}
		
		if len(andClauses) > 0 {
			return strings.Join(andClauses, " AND "), args, argIndex, nil
		}
		
	case "$not":
		notMap, ok := value.(map[string]interface{})
		if !ok {
			return "", nil, argIndex, fmt.Errorf("$not must be an object")
		}
		
		clause, args, newArgIndex, err := qt.translateQuery(notMap, argIndex)
		if err != nil {
			return "", nil, argIndex, err
		}
		
		if clause != "" {
			return "NOT (" + clause + ")", args, newArgIndex, nil
		}
	}
	
	return "", nil, argIndex, nil
}

// translateFieldOperators handles MongoDB operators on specific fields
func (qt *QueryTranslator) translateFieldOperators(field string, operators map[string]interface{}, argIndex int) (string, []interface{}, int, error) {
	conditions := []string{}
	args := []interface{}{}
	
	for op, val := range operators {
		var condition string
		var opArgs []interface{}
		
		switch op {
		case "$eq":
			placeholder := qt.getPlaceholder(argIndex)
			condition = fmt.Sprintf("%s = %s", qt.escapeField(field), placeholder)
			opArgs = []interface{}{val}
			argIndex++
			
		case "$ne":
			placeholder := qt.getPlaceholder(argIndex)
			condition = fmt.Sprintf("%s != %s", qt.escapeField(field), placeholder)
			opArgs = []interface{}{val}
			argIndex++
			
		case "$gt":
			placeholder := qt.getPlaceholder(argIndex)
			condition = fmt.Sprintf("%s > %s", qt.escapeField(field), placeholder)
			opArgs = []interface{}{val}
			argIndex++
			
		case "$gte":
			placeholder := qt.getPlaceholder(argIndex)
			condition = fmt.Sprintf("%s >= %s", qt.escapeField(field), placeholder)
			opArgs = []interface{}{val}
			argIndex++
			
		case "$lt":
			placeholder := qt.getPlaceholder(argIndex)
			condition = fmt.Sprintf("%s < %s", qt.escapeField(field), placeholder)
			opArgs = []interface{}{val}
			argIndex++
			
		case "$lte":
			placeholder := qt.getPlaceholder(argIndex)
			condition = fmt.Sprintf("%s <= %s", qt.escapeField(field), placeholder)
			opArgs = []interface{}{val}
			argIndex++
			
		case "$in":
			inValues, ok := val.([]interface{})
			if !ok {
				return "", nil, argIndex, fmt.Errorf("$in must be an array")
			}
			
			if len(inValues) == 0 {
				condition = "1=0" // Always false
			} else {
				placeholders := []string{}
				for _, v := range inValues {
					placeholders = append(placeholders, qt.getPlaceholder(argIndex))
					opArgs = append(opArgs, v)
					argIndex++
				}
				condition = fmt.Sprintf("%s IN (%s)", qt.escapeField(field), strings.Join(placeholders, ", "))
			}
			
		case "$nin":
			ninValues, ok := val.([]interface{})
			if !ok {
				return "", nil, argIndex, fmt.Errorf("$nin must be an array")
			}
			
			if len(ninValues) == 0 {
				condition = "1=1" // Always true
			} else {
				placeholders := []string{}
				for _, v := range ninValues {
					placeholders = append(placeholders, qt.getPlaceholder(argIndex))
					opArgs = append(opArgs, v)
					argIndex++
				}
				condition = fmt.Sprintf("%s NOT IN (%s)", qt.escapeField(field), strings.Join(placeholders, ", "))
			}
			
		case "$regex":
			pattern, ok := val.(string)
			if !ok {
				return "", nil, argIndex, fmt.Errorf("$regex must be a string")
			}
			
			if qt.dialect == "postgres" {
				placeholder := qt.getPlaceholder(argIndex)
				condition = fmt.Sprintf("%s ~ %s", qt.escapeField(field), placeholder)
				opArgs = []interface{}{pattern}
				argIndex++
			} else {
				// MySQL and SQLite use LIKE with % wildcards
				// Convert simple regex patterns to LIKE patterns
				likePattern := qt.regexToLike(pattern)
				placeholder := qt.getPlaceholder(argIndex)
				condition = fmt.Sprintf("%s LIKE %s", qt.escapeField(field), placeholder)
				opArgs = []interface{}{likePattern}
				argIndex++
			}
			
		case "$exists":
			exists, ok := val.(bool)
			if !ok {
				return "", nil, argIndex, fmt.Errorf("$exists must be a boolean")
			}
			
			if exists {
				condition = fmt.Sprintf("%s IS NOT NULL", qt.escapeField(field))
			} else {
				condition = fmt.Sprintf("%s IS NULL", qt.escapeField(field))
			}
			
		case "$type":
			// This is complex and database-specific
			// For now, we'll skip type checking
			continue
			
		case "$size":
			// Array size checking - requires JSON support
			size, ok := val.(float64)
			if !ok {
				if intSize, ok := val.(int); ok {
					size = float64(intSize)
				} else {
					return "", nil, argIndex, fmt.Errorf("$size must be a number")
				}
			}
			
			if qt.dialect == "postgres" {
				placeholder := qt.getPlaceholder(argIndex)
				condition = fmt.Sprintf("jsonb_array_length(%s) = %s", qt.escapeField(field), placeholder)
				opArgs = []interface{}{int(size)}
				argIndex++
			} else if qt.dialect == "mysql" {
				placeholder := qt.getPlaceholder(argIndex)
				condition = fmt.Sprintf("JSON_LENGTH(%s) = %s", qt.escapeField(field), placeholder)
				opArgs = []interface{}{int(size)}
				argIndex++
			} else {
				// SQLite with JSON1 extension
				placeholder := qt.getPlaceholder(argIndex)
				condition = fmt.Sprintf("json_array_length(%s) = %s", qt.escapeField(field), placeholder)
				opArgs = []interface{}{int(size)}
				argIndex++
			}
			
		case "$elemMatch":
			// Complex array element matching - skip for now
			continue
		}
		
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, opArgs...)
		}
	}
	
	if len(conditions) == 0 {
		return "", nil, argIndex, nil
	}
	
	// Multiple conditions on same field are ANDed
	return "(" + strings.Join(conditions, " AND ") + ")", args, argIndex, nil
}

// translateQuery is the internal version that tracks argument index
func (qt *QueryTranslator) translateQuery(query map[string]interface{}, argIndex int) (string, []interface{}, int, error) {
	conditions := []string{}
	args := []interface{}{}
	
	for field, value := range query {
		condition, newArgs, newArgIndex, err := qt.translateCondition(field, value, argIndex)
		if err != nil {
			return "", nil, argIndex, err
		}
		
		if condition != "" {
			conditions = append(conditions, condition)
			args = append(args, newArgs...)
			argIndex = newArgIndex
		}
	}
	
	if len(conditions) == 0 {
		return "", nil, argIndex, nil
	}
	
	return strings.Join(conditions, " AND "), args, argIndex, nil
}

// getPlaceholder returns the appropriate placeholder for the database
func (qt *QueryTranslator) getPlaceholder(index int) string {
	switch qt.dialect {
	case "postgres":
		return fmt.Sprintf("$%d", index)
	case "mysql":
		return "?"
	default: // sqlite
		return "?"
	}
}

// escapeField escapes field names for SQL
func (qt *QueryTranslator) escapeField(field string) string {
	// Handle nested fields (e.g., "address.city")
	parts := strings.Split(field, ".")
	
	if len(parts) > 1 && qt.supportsJSON() {
		// JSON field access
		return qt.jsonFieldAccess(parts)
	}
	
	// Simple field
	switch qt.dialect {
	case "mysql":
		return "`" + field + "`"
	case "postgres":
		return `"` + field + `"`
	default: // sqlite
		return `"` + field + `"`
	}
}

// supportsJSON checks if the database supports JSON operations
func (qt *QueryTranslator) supportsJSON() bool {
	return qt.dialect == "postgres" || qt.dialect == "mysql" || qt.dialect == "sqlite"
}

// jsonFieldAccess generates JSON field access syntax
func (qt *QueryTranslator) jsonFieldAccess(parts []string) string {
	field := parts[0]
	path := parts[1:]
	
	switch qt.dialect {
	case "postgres":
		// PostgreSQL: data->'address'->>'city'
		accessor := qt.escapeField(field)
		for i, part := range path {
			if i == len(path)-1 {
				accessor += fmt.Sprintf("->>%s", qt.quoteLiteral(part))
			} else {
				accessor += fmt.Sprintf("->%s", qt.quoteLiteral(part))
			}
		}
		return accessor
		
	case "mysql":
		// MySQL: JSON_EXTRACT(data, '$.address.city')
		jsonPath := "$"
		for _, part := range path {
			jsonPath += "." + part
		}
		return fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(%s, %s))", qt.escapeField(field), qt.quoteLiteral(jsonPath))
		
	default: // sqlite
		// SQLite: json_extract(data, '$.address.city')
		jsonPath := "$"
		for _, part := range path {
			jsonPath += "." + part
		}
		return fmt.Sprintf("json_extract(%s, %s)", qt.escapeField(field), qt.quoteLiteral(jsonPath))
	}
}

// quoteLiteral quotes a string literal for SQL
func (qt *QueryTranslator) quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// regexToLike converts simple regex patterns to SQL LIKE patterns
func (qt *QueryTranslator) regexToLike(pattern string) string {
	// This is a simplified conversion
	// Real regex is much more complex
	
	// Replace . with _
	pattern = strings.ReplaceAll(pattern, ".", "_")
	
	// Replace .* with %
	pattern = strings.ReplaceAll(pattern, ".*", "%")
	
	// Replace .+ with _%
	pattern = strings.ReplaceAll(pattern, ".+", "_%")
	
	// Add % at start and end if not present
	if !strings.HasPrefix(pattern, "^") {
		pattern = "%" + pattern
	} else {
		pattern = strings.TrimPrefix(pattern, "^")
	}
	
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "%"
	} else {
		pattern = strings.TrimSuffix(pattern, "$")
	}
	
	return pattern
}

// TranslateSort converts MongoDB sort specification to SQL ORDER BY
func (qt *QueryTranslator) TranslateSort(mongoSort interface{}) (string, error) {
	if mongoSort == nil {
		return "", nil
	}
	
	sortMap, ok := mongoSort.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("sort must be an object")
	}
	
	orderClauses := []string{}
	
	for field, direction := range sortMap {
		var dir string
		
		switch v := direction.(type) {
		case float64:
			if v > 0 {
				dir = "ASC"
			} else {
				dir = "DESC"
			}
		case int:
			if v > 0 {
				dir = "ASC"
			} else {
				dir = "DESC"
			}
		default:
			dir = "ASC"
		}
		
		orderClauses = append(orderClauses, fmt.Sprintf("%s %s", qt.escapeField(field), dir))
	}
	
	if len(orderClauses) == 0 {
		return "", nil
	}
	
	return strings.Join(orderClauses, ", "), nil
}

// TranslateProjection converts MongoDB projection to SQL SELECT fields
func (qt *QueryTranslator) TranslateProjection(mongoProjection interface{}) ([]string, error) {
	if mongoProjection == nil {
		return []string{"*"}, nil
	}
	
	projMap, ok := mongoProjection.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("projection must be an object")
	}
	
	// Check if it's inclusion or exclusion
	includeMode := false
	excludeMode := false
	
	for field, value := range projMap {
		if field == "_id" {
			continue // _id is special
		}
		
		include := false
		switch v := value.(type) {
		case float64:
			include = v > 0
		case int:
			include = v > 0
		case bool:
			include = v
		}
		
		if include {
			includeMode = true
		} else {
			excludeMode = true
		}
	}
	
	if includeMode && excludeMode {
		return nil, fmt.Errorf("cannot mix inclusion and exclusion in projection")
	}
	
	if includeMode {
		// Include only specified fields
		fields := []string{}
		
		// Always include id unless explicitly excluded
		if idVal, hasID := projMap["_id"]; hasID {
			if include, _ := idVal.(bool); !include {
				// ID explicitly excluded
			} else {
				fields = append(fields, qt.escapeField("id"))
			}
		} else {
			// ID not mentioned, include by default
			fields = append(fields, qt.escapeField("id"))
		}
		
		for field, value := range projMap {
			if field == "_id" {
				continue
			}
			
			include := false
			switch v := value.(type) {
			case float64:
				include = v > 0
			case int:
				include = v > 0
			case bool:
				include = v
			}
			
			if include {
				fields = append(fields, qt.escapeField(field))
			}
		}
		
		return fields, nil
	} else if excludeMode {
		// This is tricky - we'd need to know all fields
		// For now, return * and handle exclusion at application level
		return []string{"*"}, nil
	}
	
	return []string{"*"}, nil
}

// BuildSelectQuery builds a complete SELECT query from MongoDB-style parameters
func (qt *QueryTranslator) BuildSelectQuery(table string, mongoQuery interface{}, options map[string]interface{}) (string, []interface{}, error) {
	// Start with SELECT
	query := "SELECT "
	
	// Handle projection
	fields := []string{"*"}
	if projection, hasProjection := options["$fields"]; hasProjection {
		projFields, err := qt.TranslateProjection(projection)
		if err != nil {
			return "", nil, err
		}
		fields = projFields
	}
	query += strings.Join(fields, ", ")
	
	// FROM clause
	query += " FROM " + qt.escapeField(table)
	
	// WHERE clause
	whereClause, args, err := qt.TranslateQuery(mongoQuery)
	if err != nil {
		return "", nil, err
	}
	
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	
	// ORDER BY clause
	if sort, hasSort := options["$sort"]; hasSort {
		orderBy, err := qt.TranslateSort(sort)
		if err != nil {
			return "", nil, err
		}
		if orderBy != "" {
			query += " ORDER BY " + orderBy
		}
	}
	
	// LIMIT clause
	if limit, hasLimit := options["$limit"]; hasLimit {
		var limitVal int
		switch v := limit.(type) {
		case float64:
			limitVal = int(v)
		case int:
			limitVal = v
		}
		
		if limitVal > 0 {
			query += fmt.Sprintf(" LIMIT %d", limitVal)
		}
	}
	
	// OFFSET clause (skip)
	if skip, hasSkip := options["$skip"]; hasSkip {
		var skipVal int
		switch v := skip.(type) {
		case float64:
			skipVal = int(v)
		case int:
			skipVal = v
		}
		
		if skipVal > 0 {
			if qt.dialect == "mysql" || qt.dialect == "sqlite" {
				// MySQL and SQLite require LIMIT before OFFSET
				if _, hasLimit := options["$limit"]; !hasLimit {
					query += " LIMIT -1" // No limit
				}
			}
			query += fmt.Sprintf(" OFFSET %d", skipVal)
		}
	}
	
	return query, args, nil
}

// Example usage:
// translator := NewQueryTranslator("mysql")
// whereClause, args, err := translator.TranslateQuery(map[string]interface{}{
//     "age": map[string]interface{}{"$gte": 18},
//     "status": "active",
//     "$or": []interface{}{
//         map[string]interface{}{"role": "admin"},
//         map[string]interface{}{"permissions": map[string]interface{}{"$in": []interface{}{"write", "delete"}}},
//     },
// })
// // Result: (age >= ? AND status = ? AND ((role = ?) OR (permissions IN (?, ?))))
// // args: [18, "active", "admin", "write", "delete"]