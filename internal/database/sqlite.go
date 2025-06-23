package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDatabase implements DatabaseInterface for SQLite
type SQLiteDatabase struct {
	db     *sql.DB
	config *Config
}

// SQLiteStore implements StoreInterface for SQLite
type SQLiteStore struct {
	tableName string
	db        *sql.DB
	database  *SQLiteDatabase
}

// SQLiteUpdateResult implements UpdateResult interface
type SQLiteUpdateResult struct {
	modifiedCount int64
	upsertedCount int64
	upsertedID    interface{}
}

func (r *SQLiteUpdateResult) ModifiedCount() int64   { return r.modifiedCount }
func (r *SQLiteUpdateResult) UpsertedCount() int64   { return r.upsertedCount }
func (r *SQLiteUpdateResult) UpsertedID() interface{} { return r.upsertedID }

// SQLiteDeleteResult implements DeleteResult interface
type SQLiteDeleteResult struct {
	deletedCount int64
}

func (r *SQLiteDeleteResult) DeletedCount() int64 { return r.deletedCount }

// NewSQLiteDatabase creates a new SQLite database instance
func NewSQLiteDatabase(config *Config) (DatabaseInterface, error) {
	var dbPath string
	if config.Host == "" || config.Host == "localhost" {
		// Use file-based SQLite
		if strings.HasSuffix(config.Name, ".db") || strings.HasSuffix(config.Name, ".sqlite") {
			dbPath = config.Name
		} else {
			// Default to deployd.sqlite if no specific file extension
			if config.Name == "deployd" {
				dbPath = "deployd.sqlite"
			} else {
				dbPath = config.Name + ".sqlite"
			}
		}
	} else {
		dbPath = config.Host
	}

	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	sqliteDB := &SQLiteDatabase{
		db:     db,
		config: config,
	}

	return sqliteDB, nil
}

func (d *SQLiteDatabase) CreateStore(namespace string) StoreInterface {
	store := &SQLiteStore{
		tableName: namespace,
		db:        d.db,
		database:  d,
	}

	// Ensure table exists
	store.ensureTable()

	return store
}

func (d *SQLiteDatabase) Close() error {
	return d.db.Close()
}

func (d *SQLiteDatabase) Drop() error {
	// Get all table names
	rows, err := d.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
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
		if _, err := d.db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, table)); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

func (d *SQLiteDatabase) GetType() DatabaseType {
	return DatabaseTypeSQLite
}

// ensureTable creates the table if it doesn't exist
func (s *SQLiteStore) ensureTable() error {
	quotedTable := s.quotedTableName()
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			data JSON NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`, quotedTable)

	if _, err := s.db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table %s: %w", s.tableName, err)
	}

	// Create indexes for common queries
	indexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS "idx_%s_id" ON %s(id);
		CREATE INDEX IF NOT EXISTS "idx_%s_created_at" ON %s(created_at);
		CREATE INDEX IF NOT EXISTS "idx_%s_updated_at" ON %s(updated_at);
	`, s.tableName, quotedTable, s.tableName, quotedTable, s.tableName, quotedTable)

	if _, err := s.db.Exec(indexSQL); err != nil {
		return fmt.Errorf("failed to create indexes for table %s: %w", s.tableName, err)
	}

	return nil
}

func (s *SQLiteStore) CreateUniqueIdentifier() string {
	return generateUniqueID()
}

func (s *SQLiteStore) Insert(ctx context.Context, document interface{}) (interface{}, error) {
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

func (s *SQLiteStore) Find(ctx context.Context, query QueryBuilder, opts QueryOptions) ([]map[string]interface{}, error) {
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

func (s *SQLiteStore) FindOne(ctx context.Context, query QueryBuilder) (map[string]interface{}, error) {
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

func (s *SQLiteStore) Update(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	return s.performUpdate(ctx, query, update, false)
}

func (s *SQLiteStore) UpdateOne(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	return s.performUpdate(ctx, query, update, true)
}

func (s *SQLiteStore) performUpdate(ctx context.Context, query QueryBuilder, update UpdateBuilder, updateOne bool) (UpdateResult, error) {
	// First, find the documents to update
	existingDocs, err := s.Find(ctx, query, QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find documents to update: %w", err)
	}

	if len(existingDocs) == 0 {
		return &SQLiteUpdateResult{modifiedCount: 0}, nil
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

	return &SQLiteUpdateResult{modifiedCount: modifiedCount}, nil
}

func (s *SQLiteStore) Remove(ctx context.Context, query QueryBuilder) (DeleteResult, error) {
	// First count how many will be deleted
	count, err := s.Count(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents to delete: %w", err)
	}

	if count == 0 {
		return &SQLiteDeleteResult{deletedCount: 0}, nil
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

	return &SQLiteDeleteResult{deletedCount: count}, nil
}

func (s *SQLiteStore) Count(ctx context.Context, query QueryBuilder) (int64, error) {
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

// Specialized MongoDB-style operations (simplified implementations for SQLite)
func (s *SQLiteStore) Increment(ctx context.Context, query QueryBuilder, increments map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range increments {
		update.Inc(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *SQLiteStore) Push(ctx context.Context, query QueryBuilder, pushOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range pushOps {
		update.Push(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *SQLiteStore) Pull(ctx context.Context, query QueryBuilder, pullOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range pullOps {
		update.Pull(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *SQLiteStore) AddToSet(ctx context.Context, query QueryBuilder, addOps map[string]interface{}) (UpdateResult, error) {
	update := NewUpdateBuilder()
	for field, value := range addOps {
		update.AddToSet(field, value)
	}
	return s.Update(ctx, query, update)
}

func (s *SQLiteStore) PopFirst(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	// For SQLite, we'll implement this by updating arrays manually
	return &SQLiteUpdateResult{modifiedCount: 0}, fmt.Errorf("PopFirst not yet implemented for SQLite")
}

func (s *SQLiteStore) PopLast(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	// For SQLite, we'll implement this by updating arrays manually
	return &SQLiteUpdateResult{modifiedCount: 0}, fmt.Errorf("PopLast not yet implemented for SQLite")
}

func (s *SQLiteStore) Upsert(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
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

func (s *SQLiteStore) Aggregate(ctx context.Context, pipeline []map[string]interface{}) ([]map[string]interface{}, error) {
	// Basic aggregation support - this is a simplified implementation
	// For now, just return all documents
	query := NewQueryBuilder()
	return s.Find(ctx, query, QueryOptions{})
}

// Helper methods

func (s *SQLiteStore) quotedTableName() string {
	// Quote table names to handle special characters like hyphens
	return fmt.Sprintf(`"%s"`, s.tableName)
}

func (s *SQLiteStore) buildWhereClause(query QueryBuilder) (string, []interface{}) {
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

func (s *SQLiteStore) convertMapToSQLQuery(queryMap map[string]interface{}, builder *SQLQueryBuilder) {
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

func (s *SQLiteStore) applyFieldProjection(doc map[string]interface{}, fields map[string]int) map[string]interface{} {
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

func (s *SQLiteStore) applyUpdateOperations(doc map[string]interface{}, updateMap map[string]interface{}) {
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

func (s *SQLiteStore) toFloat64(value interface{}) (float64, bool) {
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

func (s *SQLiteStore) valuesEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

func (s *SQLiteStore) documentsEqual(a, b map[string]interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// Register SQLite database factory
func init() {
	RegisterDatabaseFactory(DatabaseTypeSQLite, NewSQLiteDatabase)
}