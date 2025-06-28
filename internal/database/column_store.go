package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hjanuschka/go-deployd/internal/metrics"
)

// ColumnStore implements StoreInterface using column-based storage
type ColumnStore struct {
	tableName     string
	db            *sql.DB
	database      DatabaseInterface
	schema        *CollectionSchema
	schemaManager *SchemaManager
}

// NewColumnStore creates a new column-based store
func NewColumnStore(tableName string, db *sql.DB, database DatabaseInterface, schemaManager *SchemaManager) (*ColumnStore, error) {
	store := &ColumnStore{
		tableName:     tableName,
		db:            db,
		database:      database,
		schemaManager: schemaManager,
	}

	// Load schema and ensure table exists
	schema, err := schemaManager.GetSchema(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	store.schema = schema

	// Ensure table schema is up to date
	if err := schemaManager.EnsureSchema(tableName); err != nil {
		return nil, fmt.Errorf("failed to ensure schema: %w", err)
	}

	return store, nil
}

// CreateUniqueIdentifier generates a unique ID
func (s *ColumnStore) CreateUniqueIdentifier() string {
	return generateUniqueID()
}

// Insert inserts a document using column-based storage
func (s *ColumnStore) Insert(ctx context.Context, document interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDatabaseOperation("insert", time.Since(start), nil)
	}()

	doc, ok := document.(map[string]interface{})
	if !ok {
		err := fmt.Errorf("document must be a map[string]interface{}")
		metrics.RecordDatabaseOperation("insert", time.Since(start), err)
		return nil, err
	}

	// Ensure the document has an ID
	if _, exists := doc["id"]; !exists {
		doc["id"] = s.CreateUniqueIdentifier()
	}

	// Add timestamps
	now := time.Now()
	if _, exists := doc["createdAt"]; !exists {
		doc["createdAt"] = now
	}
	if _, exists := doc["updatedAt"]; !exists {
		doc["updatedAt"] = now
	}

	// Separate fields into columns and JSON data
	columnValues, jsonData, err := s.separateData(doc)
	if err != nil {
		err = fmt.Errorf("failed to separate data: %w", err)
		metrics.RecordDatabaseOperation("insert", time.Since(start), err)
		return nil, err
	}

	// Build INSERT SQL
	sql, args, err := s.buildInsertSQL(columnValues, jsonData)
	if err != nil {
		err = fmt.Errorf("failed to build insert SQL: %w", err)
		metrics.RecordDatabaseOperation("insert", time.Since(start), err)
		return nil, err
	}

	// Execute insert
	_, err = s.db.ExecContext(ctx, sql, args...)
	if err != nil {
		err = fmt.Errorf("failed to insert document: %w", err)
		metrics.RecordDatabaseOperation("insert", time.Since(start), err)
		return nil, err
	}

	return doc, nil
}

// separateData separates document data into column values and JSON overflow
func (s *ColumnStore) separateData(doc map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	if !s.schema.UseColumns {
		// Use JSON-only storage
		return map[string]interface{}{
			"id":         doc["id"],
			"created_at": doc["createdAt"],
			"updated_at": doc["updatedAt"],
		}, doc, nil
	}

	columnValues := make(map[string]interface{})
	jsonData := make(map[string]interface{})

	// Create column name lookup
	columnMap := make(map[string]ColumnDefinition)
	for _, col := range s.schema.Columns {
		columnMap[col.Name] = col
	}

	for key, value := range doc {
		if col, hasColumn := columnMap[key]; hasColumn && col.Name != "data" {
			// Store in column
			convertedValue, err := s.convertValueForColumn(value, col)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert value for column %s: %w", key, err)
			}
			columnValues[key] = convertedValue
		} else {
			// Store in JSON data column
			jsonData[key] = value
		}
	}

	return columnValues, jsonData, nil
}

// convertValueForColumn converts a value to the appropriate type for a column
func (s *ColumnStore) convertValueForColumn(value interface{}, col ColumnDefinition) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch col.Type {
	case ColumnTypeText:
		return fmt.Sprintf("%v", value), nil
	case ColumnTypeInteger:
		switch v := value.(type) {
		case int:
			return int64(v), nil
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		case string:
			// Try to parse as number
			return fmt.Sprintf("%v", value), nil
		default:
			return fmt.Sprintf("%v", value), nil
		}
	case ColumnTypeReal:
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return fmt.Sprintf("%v", value), nil
		}
	case ColumnTypeBoolean:
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			return v == "true" || v == "1", nil
		case int:
			return v != 0, nil
		case float64:
			return v != 0, nil
		default:
			return false, nil
		}
	case ColumnTypeDate:
		switch v := value.(type) {
		case time.Time:
			return v, nil
		case string:
			// Try to parse as time
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return t, nil
			}
			return v, nil
		default:
			return fmt.Sprintf("%v", value), nil
		}
	case ColumnTypeJSON:
		// Convert to JSON string
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return string(jsonBytes), nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

// buildInsertSQL builds the INSERT SQL statement
func (s *ColumnStore) buildInsertSQL(columnValues map[string]interface{}, jsonData map[string]interface{}) (string, []interface{}, error) {
	quotedTable := s.quoteIdentifier(s.tableName)

	var columns []string
	var placeholders []string
	var args []interface{}

	// Add column values
	for key, value := range columnValues {
		columns = append(columns, s.quoteIdentifier(key))
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	// Add JSON data if there's any overflow or if using JSON-only mode
	if len(jsonData) > 0 {
		jsonBytes, err := json.Marshal(jsonData)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal JSON data: %w", err)
		}

		columns = append(columns, s.quoteIdentifier("data"))
		placeholders = append(placeholders, "?")
		args = append(args, string(jsonBytes))
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quotedTable,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	return sql, args, nil
}

// Find performs a column-aware query
func (s *ColumnStore) Find(ctx context.Context, query QueryBuilder, opts QueryOptions) ([]map[string]interface{}, error) {
	sql, args, err := s.buildSelectSQL(query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build select SQL: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	return s.scanRows(rows, opts.Fields)
}

// buildSelectSQL builds the SELECT SQL statement with column-aware WHERE clauses
func (s *ColumnStore) buildSelectSQL(query QueryBuilder, opts QueryOptions) (string, []interface{}, error) {
	quotedTable := s.quoteIdentifier(s.tableName)

	// Select all columns by default
	selectClause := "*"
	if len(opts.Fields) > 0 {
		selectClause = s.buildSelectClause(opts.Fields)
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", selectClause, quotedTable)
	var args []interface{}

	// Build WHERE clause using column-aware query builder
	whereClause, whereArgs := s.buildWhereClause(query)
	if whereClause != "" {
		sql += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	// Add ORDER BY with column-aware sorting
	if len(opts.Sort) > 0 {
		orderClause := s.buildOrderClause(opts.Sort)
		sql += " ORDER BY " + orderClause
	}

	// Add LIMIT and OFFSET
	if opts.Limit != nil {
		sql += fmt.Sprintf(" LIMIT %d", *opts.Limit)
	}
	if opts.Skip != nil {
		sql += fmt.Sprintf(" OFFSET %d", *opts.Skip)
	}

	return sql, args, nil
}

// buildSelectClause builds the SELECT clause for field projection
func (s *ColumnStore) buildSelectClause(fields map[string]int) string {
	if !s.schema.UseColumns {
		return "*" // For JSON-only tables, select everything
	}

	var columns []string
	hasInclusions := false

	// Check if this is inclusion or exclusion mode
	for _, include := range fields {
		if include == 1 {
			hasInclusions = true
			break
		}
	}

	if hasInclusions {
		// Include mode: only select specified fields
		columns = append(columns, s.quoteIdentifier("id")) // Always include id

		for field, include := range fields {
			if include == 1 && field != "id" {
				if s.hasColumn(field) {
					columns = append(columns, s.quoteIdentifier(field))
				}
			}
		}

		// If any non-column fields are requested, include data column
		for field, include := range fields {
			if include == 1 && !s.hasColumn(field) {
				columns = append(columns, s.quoteIdentifier("data"))
				break
			}
		}
	} else {
		// Exclusion mode: select all except specified fields
		for _, col := range s.schema.Columns {
			if exclude, exists := fields[col.Name]; !exists || exclude != 0 {
				columns = append(columns, s.quoteIdentifier(col.Name))
			}
		}
	}

	if len(columns) == 0 {
		return "*"
	}

	return strings.Join(columns, ", ")
}

// hasColumn checks if a field has a dedicated column
func (s *ColumnStore) hasColumn(fieldName string) bool {
	for _, col := range s.schema.Columns {
		if col.Name == fieldName && col.Name != "data" {
			return true
		}
	}
	return false
}

// buildWhereClause builds a column-aware WHERE clause
func (s *ColumnStore) buildWhereClause(query QueryBuilder) (string, []interface{}) {
	if sqlQuery, ok := query.(*SQLQueryBuilder); ok {
		return sqlQuery.ToSQL()
	}

	// Convert from map-based query with column awareness
	queryMap := query.ToMap()
	if len(queryMap) == 0 {
		return "", nil
	}

	sqlBuilder := NewSQLQueryBuilder()
	// Set the column checker so the SQLQueryBuilder knows which fields are columns
	sqlBuilder.SetColumnChecker(s.hasColumn)
	
	// Debug logging
	fmt.Printf("DEBUG: ColumnStore processing query: %+v\n", queryMap)
	fmt.Printf("DEBUG: ColumnStore schema columns: %+v\n", s.schema.Columns)
	fmt.Printf("DEBUG: ColumnStore UseColumns: %v\n", s.schema.UseColumns)
	
	s.convertMapToColumnSQL(queryMap, sqlBuilder)
	return sqlBuilder.ToSQL()
}

// convertMapToColumnSQL converts a MongoDB-style query to column-aware SQL
func (s *ColumnStore) convertMapToColumnSQL(queryMap map[string]interface{}, builder *SQLQueryBuilder) {
	for field, value := range queryMap {
		if field == "$or" {
			// Handle OR conditions
			if orConditions, ok := value.([]map[string]interface{}); ok {
				var orBuilders []QueryBuilder
				for _, orCond := range orConditions {
					orBuilder := NewSQLQueryBuilder()
					orBuilder.SetColumnChecker(s.hasColumn)
					s.convertMapToColumnSQL(orCond, orBuilder)
					orBuilders = append(orBuilders, orBuilder)
				}
				builder.Or(orBuilders...)
			}
		} else if strings.HasPrefix(field, "$") {
			// Skip other MongoDB operators at root level
			continue
		} else {
			// Regular field condition - use column if available
			if valueMap, ok := value.(map[string]interface{}); ok {
				// Field has operators
				for op, opValue := range valueMap {
					s.addColumnCondition(builder, field, op, opValue)
				}
			} else {
				// Simple equality
				s.addColumnCondition(builder, field, "$eq", value)
			}
		}
	}
}

// addColumnCondition adds a condition using column-aware field access
func (s *ColumnStore) addColumnCondition(builder *SQLQueryBuilder, field, op string, value interface{}) {
	// The SQLQueryBuilder now handles column vs JSON decision automatically
	// based on the column checker we set
	builder.Where(field, op, value)
}

// mongoOpToSQL converts MongoDB operators to SQL operators
func (s *ColumnStore) mongoOpToSQL(op string) string {
	switch op {
	case "$eq":
		return "="
	case "$ne":
		return "!="
	case "$gt":
		return ">"
	case "$gte":
		return ">="
	case "$lt":
		return "<"
	case "$lte":
		return "<="
	case "$in":
		return "IN"
	case "$nin":
		return "NOT IN"
	case "$regex":
		return "REGEXP"
	default:
		return "="
	}
}

// buildOrderClause builds column-aware ORDER BY clause
func (s *ColumnStore) buildOrderClause(sort map[string]int) string {
	var orderParts []string

	for field, direction := range sort {
		dir := "ASC"
		if direction == -1 {
			dir = "DESC"
		}

		if s.hasColumn(field) {
			// Use direct column
			orderParts = append(orderParts, fmt.Sprintf("%s %s", s.quoteIdentifier(field), dir))
		} else {
			// Use JSON extraction
			jsonPath := fmt.Sprintf("$.%s", field)
			switch s.database.GetType() {
			case DatabaseTypeSQLite:
				orderParts = append(orderParts, fmt.Sprintf("JSON_EXTRACT(data, '%s') %s", jsonPath, dir))
			case DatabaseTypeMySQL:
				orderParts = append(orderParts, fmt.Sprintf("JSON_EXTRACT(data, '%s') %s", jsonPath, dir))
			}
		}
	}

	return strings.Join(orderParts, ", ")
}

// scanRows scans database rows and reconstructs documents
func (s *ColumnStore) scanRows(rows *sql.Rows, fields map[string]int) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}

	for rows.Next() {
		// Create slice for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Reconstruct document
		doc := make(map[string]interface{})

		for i, columnName := range columns {
			value := values[i]

			if columnName == "data" && value != nil {
				// Merge JSON data
				if jsonStr, ok := value.(string); ok {
					var jsonData map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
						for k, v := range jsonData {
							doc[k] = v
						}
					}
				}
			} else if value != nil {
				// Direct column value
				doc[columnName] = s.convertFromDB(value)
			}
		}

		// Apply field projection if specified
		if len(fields) > 0 {
			doc = s.applyFieldProjection(doc, fields)
		}

		results = append(results, doc)
	}

	return results, nil
}

// convertFromDB converts database values to Go types
func (s *ColumnStore) convertFromDB(value interface{}) interface{} {
	switch v := value.(type) {
	case []byte:
		// Convert byte arrays to strings
		return string(v)
	case time.Time:
		return v
	default:
		return v
	}
}

// applyFieldProjection applies field projection to a document
func (s *ColumnStore) applyFieldProjection(doc map[string]interface{}, fields map[string]int) map[string]interface{} {
	result := make(map[string]interface{})

	// Check if this is inclusion or exclusion
	hasInclusions := false
	for _, include := range fields {
		if include == 1 {
			hasInclusions = true
			break
		}
	}

	if hasInclusions {
		// Inclusion mode: only include specified fields
		for field, include := range fields {
			if include == 1 {
				if value, exists := doc[field]; exists {
					result[field] = value
				}
			}
		}
		// Always include id
		if _, hasID := fields["id"]; !hasID {
			if id, exists := doc["id"]; exists {
				result["id"] = id
			}
		}
	} else {
		// Exclusion mode: include all except specified fields
		for field, value := range doc {
			if exclude, exists := fields[field]; !exists || exclude != 0 {
				result[field] = value
			}
		}
	}

	return result
}

// quoteIdentifier quotes database identifiers
func (s *ColumnStore) quoteIdentifier(name string) string {
	return s.schemaManager.quoteIdentifier(name)
}

// Implement remaining StoreInterface methods by delegating to appropriate logic...

func (s *ColumnStore) FindOne(ctx context.Context, query QueryBuilder) (map[string]interface{}, error) {
	limit := int64(1)
	opts := QueryOptions{Limit: &limit}
	results, err := s.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0], nil
}

func (s *ColumnStore) Update(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	return s.performUpdate(ctx, query, update, false)
}

func (s *ColumnStore) UpdateOne(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	return s.performUpdate(ctx, query, update, true)
}

func (s *ColumnStore) performUpdate(ctx context.Context, query QueryBuilder, update UpdateBuilder, updateOne bool) (UpdateResult, error) {
	// Implementation similar to SQLiteStore but with column awareness
	// For now, delegate to finding and updating individual documents
	existingDocs, err := s.Find(ctx, query, QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find documents to update: %w", err)
	}

	if len(existingDocs) == 0 {
		return &SQLiteUpdateResult{modifiedCount: 0}, nil
	}

	if updateOne && len(existingDocs) > 1 {
		existingDocs = existingDocs[:1]
	}

	updateMap := update.ToMap()
	modifiedCount := int64(0)

	for _, doc := range existingDocs {
		if err := s.updateSingleDocument(ctx, doc, updateMap); err != nil {
			return nil, fmt.Errorf("failed to update document: %w", err)
		}
		modifiedCount++
	}

	return &SQLiteUpdateResult{modifiedCount: modifiedCount}, nil
}

func (s *ColumnStore) updateSingleDocument(ctx context.Context, doc map[string]interface{}, updateMap map[string]interface{}) error {
	originalDoc := make(map[string]interface{})
	for k, v := range doc {
		originalDoc[k] = v
	}

	// Apply update operations
	s.applyUpdateOperations(doc, updateMap)

	// Update timestamp
	doc["updatedAt"] = time.Now()

	// Check if document actually changed
	if s.documentsEqual(originalDoc, doc) {
		return nil // No changes
	}

	// Separate data and build UPDATE SQL
	columnValues, jsonData, err := s.separateData(doc)
	if err != nil {
		return fmt.Errorf("failed to separate data: %w", err)
	}

	sql, args, err := s.buildUpdateSQL(columnValues, jsonData, doc["id"])
	if err != nil {
		return fmt.Errorf("failed to build update SQL: %w", err)
	}

	_, err = s.db.ExecContext(ctx, sql, args...)
	return err
}

func (s *ColumnStore) buildUpdateSQL(columnValues map[string]interface{}, jsonData map[string]interface{}, id interface{}) (string, []interface{}, error) {
	quotedTable := s.quoteIdentifier(s.tableName)

	var setParts []string
	var args []interface{}

	// Update column values
	for key, value := range columnValues {
		if key != "id" { // Don't update ID
			setParts = append(setParts, fmt.Sprintf("%s = ?", s.quoteIdentifier(key)))
			args = append(args, value)
		}
	}

	// Update JSON data if needed
	if len(jsonData) > 0 {
		jsonBytes, err := json.Marshal(jsonData)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal JSON data: %w", err)
		}

		setParts = append(setParts, fmt.Sprintf("%s = ?", s.quoteIdentifier("data")))
		args = append(args, string(jsonBytes))
	}

	// Add ID to WHERE clause
	args = append(args, id)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		quotedTable,
		strings.Join(setParts, ", "),
		s.quoteIdentifier("id"))

	return sql, args, nil
}

// Helper methods from SQLiteStore
func (s *ColumnStore) applyUpdateOperations(doc map[string]interface{}, updateMap map[string]interface{}) {
	for operation, fields := range updateMap {
		if fieldMap, ok := fields.(map[string]interface{}); ok {
			switch operation {
			case "$set":
				for field, value := range fieldMap {
					doc[field] = value
				}
			case "$unset":
				for field := range fieldMap {
					delete(doc, field)
				}
			case "$inc":
				for field, value := range fieldMap {
					if existing, exists := doc[field]; exists {
						if existingNum, ok := s.toFloat64(existing); ok {
							if incNum, ok := s.toFloat64(value); ok {
								doc[field] = existingNum + incNum
							}
						}
					} else {
						doc[field] = value
					}
				}
			case "$push":
				for field, value := range fieldMap {
					if existing, exists := doc[field]; exists {
						if existingArray, ok := existing.([]interface{}); ok {
							doc[field] = append(existingArray, value)
						}
					} else {
						doc[field] = []interface{}{value}
					}
				}
			case "$pull":
				for field, value := range fieldMap {
					if existing, exists := doc[field]; exists {
						if existingArray, ok := existing.([]interface{}); ok {
							var newArray []interface{}
							for _, item := range existingArray {
								if !s.valuesEqual(item, value) {
									newArray = append(newArray, item)
								}
							}
							doc[field] = newArray
						}
					}
				}
			case "$addToSet":
				for field, value := range fieldMap {
					if existing, exists := doc[field]; exists {
						if existingArray, ok := existing.([]interface{}); ok {
							// Check if value already exists
							exists := false
							for _, item := range existingArray {
								if s.valuesEqual(item, value) {
									exists = true
									break
								}
							}
							if !exists {
								doc[field] = append(existingArray, value)
							}
						}
					} else {
						doc[field] = []interface{}{value}
					}
				}
			}
		}
	}
}

func (s *ColumnStore) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func (s *ColumnStore) valuesEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

func (s *ColumnStore) documentsEqual(a, b map[string]interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// Additional StoreInterface methods to implement...
func (s *ColumnStore) Remove(ctx context.Context, query QueryBuilder) (DeleteResult, error) {
	// Implementation similar to SQLiteStore.Remove
	return &SQLiteDeleteResult{deletedCount: 0}, fmt.Errorf("Remove not yet implemented for ColumnStore")
}

func (s *ColumnStore) Count(ctx context.Context, query QueryBuilder) (int64, error) {
	// Implementation similar to SQLiteStore.Count
	return 0, fmt.Errorf("Count not yet implemented for ColumnStore")
}

// Additional MongoDB-style operations...
func (s *ColumnStore) Increment(ctx context.Context, query QueryBuilder, increments map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range increments {
		update.Inc(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *ColumnStore) Push(ctx context.Context, query QueryBuilder, pushOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range pushOps {
		update.Push(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *ColumnStore) Pull(ctx context.Context, query QueryBuilder, pullOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range pullOps {
		update.Pull(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *ColumnStore) AddToSet(ctx context.Context, query QueryBuilder, addOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range addOps {
		update.AddToSet(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *ColumnStore) PopFirst(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	return &SQLiteUpdateResult{modifiedCount: 0}, fmt.Errorf("PopFirst not yet implemented for ColumnStore")
}

func (s *ColumnStore) PopLast(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	return &SQLiteUpdateResult{modifiedCount: 0}, fmt.Errorf("PopLast not yet implemented for ColumnStore")
}

func (s *ColumnStore) Upsert(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	// Try update first
	result, err := s.Update(ctx, query, update)
	if err != nil {
		return nil, err
	}

	if result.ModifiedCount() > 0 {
		return result, nil
	}

	// If no documents were updated, create a new one
	updateMap := update.ToMap()
	queryMap := query.ToMap()

	// Merge query and update into a new document
	newDoc := make(map[string]interface{})

	// Add query fields
	for field, value := range queryMap {
		if !strings.HasPrefix(field, "$") {
			newDoc[field] = value
		}
	}

	// Apply update operations to create the document
	s.applyUpdateOperations(newDoc, updateMap)

	// Insert the new document
	_, err = s.Insert(ctx, newDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert document: %w", err)
	}

	return &SQLiteUpdateResult{
		modifiedCount: 0,
		upsertedCount: 1,
		upsertedID:    newDoc["id"],
	}, nil
}

func (s *ColumnStore) Aggregate(ctx context.Context, pipeline []map[string]interface{}) ([]map[string]interface{}, error) {
	// Basic aggregation support
	query := NewQueryBuilder()
	return s.Find(ctx, query, QueryOptions{})
}

// Enhanced MongoDB-style query methods
func (s *ColumnStore) FindWithRawQuery(ctx context.Context, mongoQuery interface{}, options map[string]interface{}) ([]map[string]interface{}, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	// Convert to QueryBuilder and use existing Find method
	queryBuilder := NewQueryBuilder()
	for field, value := range parsedQuery {
		if !strings.HasPrefix(field, "$") {
			queryBuilder.Where(field, "=", value)
		}
	}

	// Convert options to QueryOptions
	queryOpts := QueryOptions{}
	
	if sort, exists := options["$sort"]; exists {
		if sortMap, ok := sort.(map[string]interface{}); ok {
			queryOpts.Sort = make(map[string]int)
			for field, direction := range sortMap {
				if dir, ok := direction.(int); ok {
					queryOpts.Sort[field] = dir
				}
			}
		}
	}

	if limit, exists := options["$limit"]; exists {
		if limitInt, ok := limit.(int); ok {
			limit64 := int64(limitInt)
			queryOpts.Limit = &limit64
		}
	}

	if skip, exists := options["$skip"]; exists {
		if skipInt, ok := skip.(int); ok {
			skip64 := int64(skipInt)
			queryOpts.Skip = &skip64
		}
	}

	if fields, exists := options["$fields"]; exists {
		if fieldsMap, ok := fields.(map[string]interface{}); ok {
			queryOpts.Fields = make(map[string]int)
			for field, include := range fieldsMap {
				if inc, ok := include.(int); ok {
					queryOpts.Fields[field] = inc
				} else if inc, ok := include.(bool); ok && inc {
					queryOpts.Fields[field] = 1
				}
			}
		}
	}

	return s.Find(ctx, queryBuilder, queryOpts)
}

func (s *ColumnStore) CountWithRawQuery(ctx context.Context, mongoQuery interface{}) (int64, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return 0, err
	}

	// Convert to QueryBuilder and use existing Count method
	queryBuilder := NewQueryBuilder()
	for field, value := range parsedQuery {
		if !strings.HasPrefix(field, "$") {
			queryBuilder.Where(field, "=", value)
		}
	}

	return s.Count(ctx, queryBuilder)
}

func (s *ColumnStore) UpdateWithRawQuery(ctx context.Context, mongoQuery interface{}, mongoUpdate interface{}) (UpdateResult, error) {
	// Parse the MongoDB query and update
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	parsedUpdate, err := ParseMongoQuery(mongoUpdate)
	if err != nil {
		return nil, err
	}

	// Convert to QueryBuilder
	queryBuilder := NewQueryBuilder()
	for field, value := range parsedQuery {
		if !strings.HasPrefix(field, "$") {
			queryBuilder.Where(field, "=", value)
		}
	}

	// Convert to UpdateBuilder
	updateBuilder := NewUpdateBuilder()
	if setOps, exists := parsedUpdate["$set"]; exists {
		if setMap, ok := setOps.(map[string]interface{}); ok {
			for field, value := range setMap {
				updateBuilder.Set(field, value)
			}
		}
	}

	return s.Update(ctx, queryBuilder, updateBuilder)
}

func (s *ColumnStore) RemoveWithRawQuery(ctx context.Context, mongoQuery interface{}) (DeleteResult, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	// Convert to QueryBuilder and use existing Remove method
	queryBuilder := NewQueryBuilder()
	for field, value := range parsedQuery {
		if !strings.HasPrefix(field, "$") {
			queryBuilder.Where(field, "=", value)
		}
	}

	return s.Remove(ctx, queryBuilder)
}
