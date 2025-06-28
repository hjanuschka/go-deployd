package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDatabase implements DatabaseInterface for MySQL
type MySQLDatabase struct {
	db            *sql.DB
	config        *Config
	schemaManager *SchemaManager
}

// MySQLStore implements StoreInterface for MySQL
type MySQLStore struct {
	tableName string
	db        *sql.DB
	database  *MySQLDatabase
}

// MySQLUpdateResult implements UpdateResult interface
type MySQLUpdateResult struct {
	modifiedCount int64
	upsertedCount int64
	upsertedID    interface{}
}

func (r *MySQLUpdateResult) ModifiedCount() int64    { return r.modifiedCount }
func (r *MySQLUpdateResult) UpsertedCount() int64    { return r.upsertedCount }
func (r *MySQLUpdateResult) UpsertedID() interface{} { return r.upsertedID }

// MySQLDeleteResult implements DeleteResult interface
type MySQLDeleteResult struct {
	deletedCount int64
}

func (r *MySQLDeleteResult) DeletedCount() int64 { return r.deletedCount }

// NewMySQLDatabase creates a new MySQL database instance
func NewMySQLDatabase(config *Config) (DatabaseInterface, error) {
	// Build MySQL connection string
	var dsn string
	if config.Username != "" && config.Password != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
			config.Username, config.Password, config.Host, config.Port, config.Name)
	} else if config.Username != "" {
		dsn = fmt.Sprintf("%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
			config.Username, config.Host, config.Port, config.Name)
	} else {
		dsn = fmt.Sprintf("tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
			config.Host, config.Port, config.Name)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(50) // Increased from 25
	db.SetMaxIdleConns(10) // Increased from 5
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute) // Add idle timeout

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	mysqlDB := &MySQLDatabase{
		db:     db,
		config: config,
	}

	// Initialize schema manager
	mysqlDB.schemaManager = NewSchemaManager(db, DatabaseTypeMySQL, "")

	return mysqlDB, nil
}

func (d *MySQLDatabase) CreateStore(namespace string) StoreInterface {
	// Check if this collection should use column-based storage
	schema, err := d.schemaManager.GetSchema(namespace)
	if err != nil {
		// Log error but fall back to JSON store
		fmt.Printf("Warning: failed to get schema for %s, using JSON storage: %v\n", namespace, err)
		store := &MySQLStore{
			tableName: namespace,
			db:        d.db,
			database:  d,
		}
		store.ensureTable()
		return store
	}

	if schema.UseColumns {
		// Use column-based storage
		columnStore, err := NewColumnStore(namespace, d.db, d, d.schemaManager)
		if err != nil {
			// Log error but fall back to JSON store
			fmt.Printf("Warning: failed to create column store for %s, using JSON storage: %v\n", namespace, err)
			store := &MySQLStore{
				tableName: namespace,
				db:        d.db,
				database:  d,
			}
			store.ensureTable()
			return store
		}
		return columnStore
	}

	// Use traditional JSON-based storage
	store := &MySQLStore{
		tableName: namespace,
		db:        d.db,
		database:  d,
	}

	// Ensure table exists
	store.ensureTable()

	return store
}

func (d *MySQLDatabase) Close() error {
	return d.db.Close()
}

func (d *MySQLDatabase) Drop() error {
	// Get all table names from information_schema
	rows, err := d.db.Query("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'")
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Drop all tables
	for _, table := range tables {
		if _, err := d.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

func (d *MySQLDatabase) GetType() DatabaseType {
	return DatabaseTypeMySQL
}

// ensureTable creates the table if it doesn't exist
func (s *MySQLStore) ensureTable() error {
	quotedTable := s.quotedTableName()
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			data JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_created_at (created_at),
			INDEX idx_updated_at (updated_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, quotedTable)

	if _, err := s.db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table %s: %w", s.tableName, err)
	}

	return nil
}

func (s *MySQLStore) CreateUniqueIdentifier() string {
	return generateUniqueID()
}

func (s *MySQLStore) Insert(ctx context.Context, document interface{}) (interface{}, error) {
	doc, ok := document.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("document must be a map[string]interface{}")
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

	// Serialize to JSON
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	insertSQL := fmt.Sprintf("INSERT INTO %s (id, data, created_at, updated_at) VALUES (?, ?, ?, ?)", s.quotedTableName())
	_, err = s.db.ExecContext(ctx, insertSQL, doc["id"], string(jsonData), now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	return doc, nil
}

func (s *MySQLStore) Find(ctx context.Context, query QueryBuilder, opts QueryOptions) ([]map[string]interface{}, error) {
	baseSQL := fmt.Sprintf("SELECT data FROM %s", s.quotedTableName())
	var args []interface{}

	// Build WHERE clause
	whereClause, whereArgs := s.buildWhereClause(query)
	if whereClause != "" {
		baseSQL += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	// Add ORDER BY
	if len(opts.Sort) > 0 {
		var orderParts []string
		for field, direction := range opts.Sort {
			dir := "ASC"
			if direction == -1 {
				dir = "DESC"
			}
			orderParts = append(orderParts, fmt.Sprintf("JSON_EXTRACT(data, '$.%s') %s", field, dir))
		}
		baseSQL += " ORDER BY " + strings.Join(orderParts, ", ")
	}

	// Add LIMIT and OFFSET
	if opts.Limit != nil {
		baseSQL += fmt.Sprintf(" LIMIT %d", *opts.Limit)
	}
	if opts.Skip != nil {
		baseSQL += fmt.Sprintf(" OFFSET %d", *opts.Skip)
	}

	rows, err := s.db.QueryContext(ctx, baseSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var jsonData string
		if err := rows.Scan(&jsonData); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &doc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document: %w", err)
		}

		// Apply field projection if specified
		if len(opts.Fields) > 0 {
			doc = s.applyFieldProjection(doc, opts.Fields)
		}

		results = append(results, doc)
	}

	return results, nil
}

func (s *MySQLStore) FindOne(ctx context.Context, query QueryBuilder) (map[string]interface{}, error) {
	opts := QueryOptions{Limit: &[]int64{1}[0]}
	results, err := s.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0], nil
}

func (s *MySQLStore) Update(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	return s.performUpdate(ctx, query, update, false)
}

func (s *MySQLStore) UpdateOne(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	return s.performUpdate(ctx, query, update, true)
}

func (s *MySQLStore) performUpdate(ctx context.Context, query QueryBuilder, update UpdateBuilder, updateOne bool) (UpdateResult, error) {
	// First, find the documents to update
	existingDocs, err := s.Find(ctx, query, QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find documents to update: %w", err)
	}

	if len(existingDocs) == 0 {
		return &MySQLUpdateResult{modifiedCount: 0}, nil
	}

	// If updateOne is true, only update the first document
	if updateOne && len(existingDocs) > 1 {
		existingDocs = existingDocs[:1]
	}

	updateMap := update.ToMap()
	modifiedCount := int64(0)

	for _, doc := range existingDocs {
		originalDoc := make(map[string]interface{})
		for k, v := range doc {
			originalDoc[k] = v
		}

		// Apply update operations
		s.applyUpdateOperations(doc, updateMap)

		// Update timestamp
		doc["updatedAt"] = time.Now()

		// Check if document actually changed
		if !s.documentsEqual(originalDoc, doc) {
			// Serialize and update in database
			jsonData, err := json.Marshal(doc)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal updated document: %w", err)
			}

			updateSQL := fmt.Sprintf("UPDATE %s SET data = ?, updated_at = ? WHERE id = ?", s.quotedTableName())
			_, err = s.db.ExecContext(ctx, updateSQL, string(jsonData), doc["updatedAt"], doc["id"])
			if err != nil {
				return nil, fmt.Errorf("failed to update document: %w", err)
			}

			modifiedCount++
		}
	}

	return &MySQLUpdateResult{modifiedCount: modifiedCount}, nil
}

func (s *MySQLStore) Remove(ctx context.Context, query QueryBuilder) (DeleteResult, error) {
	// First count how many will be deleted
	count, err := s.Count(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents to delete: %w", err)
	}

	if count == 0 {
		return &MySQLDeleteResult{deletedCount: 0}, nil
	}

	// Build DELETE query
	deleteSQL := fmt.Sprintf("DELETE FROM %s", s.quotedTableName())
	var args []interface{}

	whereClause, whereArgs := s.buildWhereClause(query)
	if whereClause != "" {
		deleteSQL += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	_, err = s.db.ExecContext(ctx, deleteSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to delete documents: %w", err)
	}

	return &MySQLDeleteResult{deletedCount: count}, nil
}

func (s *MySQLStore) Count(ctx context.Context, query QueryBuilder) (int64, error) {
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.quotedTableName())
	var args []interface{}

	whereClause, whereArgs := s.buildWhereClause(query)
	if whereClause != "" {
		countSQL += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	var count int64
	err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// Specialized MongoDB-style operations
func (s *MySQLStore) Increment(ctx context.Context, query QueryBuilder, increments map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range increments {
		update.Inc(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *MySQLStore) Push(ctx context.Context, query QueryBuilder, pushOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range pushOps {
		update.Push(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *MySQLStore) Pull(ctx context.Context, query QueryBuilder, pullOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range pullOps {
		update.Pull(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *MySQLStore) AddToSet(ctx context.Context, query QueryBuilder, addOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range addOps {
		update.AddToSet(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *MySQLStore) PopFirst(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	// For MySQL, we'll implement this by updating arrays manually
	return &MySQLUpdateResult{modifiedCount: 0}, fmt.Errorf("PopFirst not yet implemented for MySQL")
}

func (s *MySQLStore) PopLast(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	// For MySQL, we'll implement this by updating arrays manually
	return &MySQLUpdateResult{modifiedCount: 0}, fmt.Errorf("PopLast not yet implemented for MySQL")
}

func (s *MySQLStore) Upsert(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
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

	return &MySQLUpdateResult{
		modifiedCount: 0,
		upsertedCount: 1,
		upsertedID:    newDoc["id"],
	}, nil
}

func (s *MySQLStore) Aggregate(ctx context.Context, pipeline []map[string]interface{}) ([]map[string]interface{}, error) {
	// Basic aggregation support - this is a simplified implementation
	// For now, just return all documents
	query := NewQueryBuilder()
	return s.Find(ctx, query, QueryOptions{})
}

// Helper methods

func (s *MySQLStore) quotedTableName() string {
	// Quote table names with backticks for MySQL
	return fmt.Sprintf("`%s`", s.tableName)
}

func (s *MySQLStore) buildWhereClause(query QueryBuilder) (string, []interface{}) {
	if sqlQuery, ok := query.(*SQLQueryBuilder); ok {
		return sqlQuery.ToSQL()
	}

	// Convert from map-based query
	queryMap := query.ToMap()
	if len(queryMap) == 0 {
		return "", nil
	}

	sqlBuilder := NewSQLQueryBuilder()
	s.convertMapToSQLQuery(queryMap, sqlBuilder)
	return sqlBuilder.ToSQL()
}

func (s *MySQLStore) convertMapToSQLQuery(queryMap map[string]interface{}, builder *SQLQueryBuilder) {
	for field, value := range queryMap {
		if field == "$or" {
			// Handle OR conditions
			if orConditions, ok := value.([]map[string]interface{}); ok {
				var orBuilders []QueryBuilder
				for _, orCond := range orConditions {
					orBuilder := NewSQLQueryBuilder()
					s.convertMapToSQLQuery(orCond, orBuilder)
					orBuilders = append(orBuilders, orBuilder)
				}
				builder.Or(orBuilders...)
			}
		} else if strings.HasPrefix(field, "$") {
			// Skip other MongoDB operators at root level
			continue
		} else {
			// Regular field condition
			if valueMap, ok := value.(map[string]interface{}); ok {
				// Field has operators
				for op, opValue := range valueMap {
					builder.Where(field, op, opValue)
				}
			} else {
				// Simple equality
				builder.Where(field, "$eq", value)
			}
		}
	}
}

func (s *MySQLStore) applyFieldProjection(doc map[string]interface{}, fields map[string]int) map[string]interface{} {
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

func (s *MySQLStore) applyUpdateOperations(doc map[string]interface{}, updateMap map[string]interface{}) {
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

func (s *MySQLStore) toFloat64(value interface{}) (float64, bool) {
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

func (s *MySQLStore) valuesEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

func (s *MySQLStore) documentsEqual(a, b map[string]interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// Enhanced MongoDB-style query methods
func (s *MySQLStore) FindWithRawQuery(ctx context.Context, mongoQuery interface{}, options map[string]interface{}) ([]map[string]interface{}, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	// Use the query translator to convert MongoDB query to SQL
	translator := NewQueryTranslator("mysql")
	whereClause, args, err := translator.TranslateQuery(parsedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to translate query: %w", err)
	}

	// Build the SQL query
	query := fmt.Sprintf("SELECT * FROM %s", s.quotedTableName())
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	// Add sorting
	if sort, exists := options["$sort"]; exists {
		if sortMap, ok := sort.(map[string]interface{}); ok {
			orderBy, err := translator.TranslateSort(sortMap)
			if err == nil && orderBy != "" {
				query += " ORDER BY " + orderBy
			}
		}
	}

	// Add limit and offset
	if limit, exists := options["$limit"]; exists {
		if limitInt, ok := limit.(int); ok {
			query += fmt.Sprintf(" LIMIT %d", limitInt)
		}
	}

	if skip, exists := options["$skip"]; exists {
		if skipInt, ok := skip.(int); ok {
			query += fmt.Sprintf(" OFFSET %d", skipInt)
		}
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice to hold the column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Create a map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// Try to parse JSON fields
				var jsonVal interface{}
				if err := json.Unmarshal(b, &jsonVal); err == nil {
					row[col] = jsonVal
				} else {
					row[col] = string(b)
				}
			} else {
				row[col] = val
			}
		}

		// Apply field projection if specified
		if fields, exists := options["$fields"]; exists {
			if fieldsMap, ok := fields.(map[string]interface{}); ok {
				projectedRow := make(map[string]interface{})
				for field, include := range fieldsMap {
					if include == 1 || include == true {
						if val, exists := row[field]; exists {
							projectedRow[field] = val
						}
					}
				}
				// Always include _id if not explicitly excluded
				if _, excluded := fieldsMap["id"]; !excluded {
					if id, exists := row["id"]; exists {
						projectedRow["id"] = id
					}
				}
				row = projectedRow
			}
		}

		results = append(results, row)
	}

	return results, rows.Err()
}

func (s *MySQLStore) CountWithRawQuery(ctx context.Context, mongoQuery interface{}) (int64, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return 0, err
	}

	// Use the query translator to convert MongoDB query to SQL
	translator := NewQueryTranslator("mysql")
	whereClause, args, err := translator.TranslateQuery(parsedQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to translate query: %w", err)
	}

	// Build the count query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.quotedTableName())
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int64
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (s *MySQLStore) UpdateWithRawQuery(ctx context.Context, mongoQuery interface{}, mongoUpdate interface{}) (UpdateResult, error) {
	// Parse the MongoDB query and update
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	parsedUpdate, err := ParseMongoQuery(mongoUpdate)
	if err != nil {
		return nil, err
	}

	// Use the query translator to convert MongoDB query to SQL
	translator := NewQueryTranslator("mysql")
	whereClause, args, err := translator.TranslateQuery(parsedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to translate query: %w", err)
	}

	// Build the update query
	var setParts []string
	var updateArgs []interface{}

	// Handle $set operations
	if setOps, exists := parsedUpdate["$set"]; exists {
		if setMap, ok := setOps.(map[string]interface{}); ok {
			for field, value := range setMap {
				setParts = append(setParts, fmt.Sprintf("`%s` = ?", field))
				// Convert complex types to JSON
				if jsonData, err := json.Marshal(value); err == nil && (fmt.Sprintf("%T", value) == "map[string]interface {}" || fmt.Sprintf("%T", value) == "[]interface {}") {
					updateArgs = append(updateArgs, string(jsonData))
				} else {
					updateArgs = append(updateArgs, value)
				}
			}
		}
	}

	if len(setParts) == 0 {
		return &MySQLUpdateResult{}, nil
	}

	query := fmt.Sprintf("UPDATE %s SET %s", s.quotedTableName(), strings.Join(setParts, ", "))
	if whereClause != "" {
		query += " WHERE " + whereClause
		updateArgs = append(updateArgs, args...)
	}

	result, err := s.db.ExecContext(ctx, query, updateArgs...)
	if err != nil {
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()
	return &MySQLUpdateResult{modifiedCount: rowsAffected}, nil
}

func (s *MySQLStore) RemoveWithRawQuery(ctx context.Context, mongoQuery interface{}) (DeleteResult, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	// Use the query translator to convert MongoDB query to SQL
	translator := NewQueryTranslator("mysql")
	whereClause, args, err := translator.TranslateQuery(parsedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to translate query: %w", err)
	}

	// Build the delete query
	query := fmt.Sprintf("DELETE FROM %s", s.quotedTableName())
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()
	return &MySQLDeleteResult{deletedCount: rowsAffected}, nil
}

// Register MySQL database factory
func init() {
	RegisterDatabaseFactory(DatabaseTypeMySQL, NewMySQLDatabase)
}
