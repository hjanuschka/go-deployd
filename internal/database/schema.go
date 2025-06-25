package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ColumnType represents the SQL column type for a field
type ColumnType string

const (
	ColumnTypeText    ColumnType = "TEXT"
	ColumnTypeInteger ColumnType = "INTEGER"
	ColumnTypeReal    ColumnType = "REAL"
	ColumnTypeBoolean ColumnType = "BOOLEAN"
	ColumnTypeDate    ColumnType = "DATETIME"
	ColumnTypeJSON    ColumnType = "JSON"
)

// FieldDefinition represents a field definition from config.json
type FieldDefinition struct {
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default"`
	Index    bool        `json:"index"`
}

// ColumnDefinition represents a database column
type ColumnDefinition struct {
	Name         string
	Type         ColumnType
	Required     bool
	Default      interface{}
	Index        bool
	IsPrimary    bool
	IsTimestamp  bool
	OriginalType string // Original deployd type (string, number, etc.)
}

// CollectionSchema represents the schema for a collection
type CollectionSchema struct {
	Name       string
	Columns    []ColumnDefinition
	UseColumns bool // Whether to use column-based or JSON-based storage
	ConfigPath string
	ModTime    time.Time
}

// SchemaManager handles schema detection, migration, and management
type SchemaManager struct {
	db         *sql.DB
	dbType     DatabaseType
	configPath string
	schemas    map[string]*CollectionSchema
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(db *sql.DB, dbType DatabaseType, configPath string) *SchemaManager {
	return &SchemaManager{
		db:         db,
		dbType:     dbType,
		configPath: configPath,
		schemas:    make(map[string]*CollectionSchema),
	}
}

// GetSchema returns the schema for a collection, loading it if necessary
func (sm *SchemaManager) GetSchema(collectionName string) (*CollectionSchema, error) {
	// Check if we have a cached schema
	if schema, exists := sm.schemas[collectionName]; exists {
		// Check if config file has been modified
		configPath := sm.getConfigPath(collectionName)
		if stat, err := os.Stat(configPath); err == nil {
			if stat.ModTime().After(schema.ModTime) {
				// Config file was modified, reload schema
				delete(sm.schemas, collectionName)
			} else {
				return schema, nil
			}
		}
	}

	// Load schema from config
	schema, err := sm.loadSchemaFromConfig(collectionName)
	if err != nil {
		return nil, err
	}

	sm.schemas[collectionName] = schema
	return schema, nil
}

// loadSchemaFromConfig loads schema definition from config.json
func (sm *SchemaManager) loadSchemaFromConfig(collectionName string) (*CollectionSchema, error) {
	configPath := sm.getConfigPath(collectionName)
	
	// Check if config file exists
	stat, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		// No config file, use default JSON-based schema
		return &CollectionSchema{
			Name:       collectionName,
			UseColumns: false,
			Columns:    sm.getDefaultColumns(),
			ConfigPath: configPath,
			ModTime:    time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Read config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Properties map[string]FieldDefinition `json:"properties"`
		Options    struct {
			UseColumns bool `json:"useColumns"`
		} `json:"options"`
	}

	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Convert field definitions to column definitions
	columns := sm.getDefaultColumns() // Always include id, created_at, updated_at
	
	for fieldName, fieldDef := range config.Properties {
		// Skip system fields that are already included
		if fieldName == "id" || fieldName == "createdAt" || fieldName == "updatedAt" {
			continue
		}

		column := ColumnDefinition{
			Name:         fieldName,
			Type:         sm.mapFieldTypeToColumn(fieldDef.Type),
			Required:     fieldDef.Required,
			Default:      fieldDef.Default,
			Index:        fieldDef.Index,
			OriginalType: fieldDef.Type,
		}

		columns = append(columns, column)
	}

	return &CollectionSchema{
		Name:       collectionName,
		UseColumns: config.Options.UseColumns,
		Columns:    columns,
		ConfigPath: configPath,
		ModTime:    stat.ModTime(),
	}, nil
}

// getConfigPath returns the path to the config.json file for a collection
func (sm *SchemaManager) getConfigPath(collectionName string) string {
	if sm.configPath != "" {
		return filepath.Join(sm.configPath, collectionName, "config.json")
	}
	return filepath.Join("resources", collectionName, "config.json")
}

// getDefaultColumns returns the default system columns
func (sm *SchemaManager) getDefaultColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{
			Name:        "id",
			Type:        ColumnTypeText,
			Required:    true,
			IsPrimary:   true,
			OriginalType: "string",
		},
		{
			Name:        "created_at",
			Type:        ColumnTypeDate,
			Required:    true,
			IsTimestamp: true,
			OriginalType: "date",
		},
		{
			Name:        "updated_at",
			Type:        ColumnTypeDate,
			Required:    true,
			IsTimestamp: true,
			OriginalType: "date",
		},
		{
			Name:        "data",
			Type:        ColumnTypeJSON,
			Required:    false,
			OriginalType: "object",
		},
	}
}

// mapFieldTypeToColumn maps deployd field types to SQL column types
func (sm *SchemaManager) mapFieldTypeToColumn(fieldType string) ColumnType {
	switch strings.ToLower(fieldType) {
	case "string", "text":
		return ColumnTypeText
	case "number", "integer", "int":
		return ColumnTypeReal
	case "boolean", "bool":
		return ColumnTypeBoolean
	case "date", "datetime", "timestamp":
		return ColumnTypeDate
	case "array", "object":
		return ColumnTypeJSON
	default:
		return ColumnTypeText // Default to text for unknown types
	}
}

// EnsureSchema ensures the table schema matches the collection config
func (sm *SchemaManager) EnsureSchema(collectionName string) error {
	schema, err := sm.GetSchema(collectionName)
	if err != nil {
		return err
	}

	// Check if table exists
	exists, err := sm.tableExists(collectionName)
	if err != nil {
		return err
	}

	if !exists {
		return sm.createTable(schema)
	}

	// Table exists, check if schema needs updating
	if schema.UseColumns {
		return sm.migrateSchema(schema)
	}

	return nil
}

// tableExists checks if a table exists
func (sm *SchemaManager) tableExists(tableName string) (bool, error) {
	var query string
	var args []interface{}

	switch sm.dbType {
	case DatabaseTypeSQLite:
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		args = []interface{}{tableName}
	case DatabaseTypeMySQL:
		query = "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?"
		args = []interface{}{tableName}
	default:
		return false, fmt.Errorf("unsupported database type: %v", sm.dbType)
	}

	var name string
	err := sm.db.QueryRow(query, args...).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// createTable creates a new table with the specified schema
func (sm *SchemaManager) createTable(schema *CollectionSchema) error {
	var sql strings.Builder
	quotedTable := sm.quoteIdentifier(schema.Name)
	
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (", quotedTable))

	for i, column := range schema.Columns {
		if i > 0 {
			sql.WriteString(", ")
		}

		sql.WriteString(sm.buildColumnDefinition(column))
	}

	sql.WriteString(")")

	if _, err := sm.db.Exec(sql.String()); err != nil {
		return fmt.Errorf("failed to create table %s: %w", schema.Name, err)
	}

	// Create indexes
	for _, column := range schema.Columns {
		if column.Index && !column.IsPrimary {
			if err := sm.createIndex(schema.Name, column.Name); err != nil {
				// Log warning but don't fail
				fmt.Printf("Warning: failed to create index for %s.%s: %v\n", schema.Name, column.Name, err)
			}
		}
	}

	return nil
}

// buildColumnDefinition builds the SQL column definition
func (sm *SchemaManager) buildColumnDefinition(column ColumnDefinition) string {
	var def strings.Builder
	
	def.WriteString(sm.quoteIdentifier(column.Name))
	def.WriteString(" ")
	def.WriteString(sm.getColumnTypeSQL(column.Type))

	if column.IsPrimary {
		def.WriteString(" PRIMARY KEY")
	}

	if column.Required && !column.IsPrimary {
		def.WriteString(" NOT NULL")
	}

	if column.Default != nil && !column.IsTimestamp {
		switch v := column.Default.(type) {
		case string:
			if v == "now" && column.Type == ColumnTypeDate {
				def.WriteString(" DEFAULT CURRENT_TIMESTAMP")
			} else {
				def.WriteString(fmt.Sprintf(" DEFAULT '%s'", v))
			}
		case bool:
			if v {
				def.WriteString(" DEFAULT 1")
			} else {
				def.WriteString(" DEFAULT 0")
			}
		default:
			def.WriteString(fmt.Sprintf(" DEFAULT %v", v))
		}
	} else if column.IsTimestamp {
		def.WriteString(" DEFAULT CURRENT_TIMESTAMP")
	}

	return def.String()
}

// getColumnTypeSQL returns the database-specific SQL type for a column type
func (sm *SchemaManager) getColumnTypeSQL(colType ColumnType) string {
	switch sm.dbType {
	case DatabaseTypeMySQL:
		switch colType {
		case ColumnTypeText:
			return "VARCHAR(255)"
		case ColumnTypeInteger:
			return "BIGINT"
		case ColumnTypeReal:
			return "DOUBLE"
		case ColumnTypeBoolean:
			return "TINYINT(1)"
		case ColumnTypeDate:
			return "DATETIME"
		case ColumnTypeJSON:
			return "JSON"
		default:
			return "VARCHAR(255)"
		}
	case DatabaseTypeSQLite:
		// SQLite is flexible with types, use the generic names
		return string(colType)
	default:
		return string(colType)
	}
}

// quoteIdentifier quotes an SQL identifier (table or column name)
func (sm *SchemaManager) quoteIdentifier(name string) string {
	switch sm.dbType {
	case DatabaseTypeSQLite:
		return fmt.Sprintf(`"%s"`, name)
	case DatabaseTypeMySQL:
		return fmt.Sprintf("`%s`", name)
	default:
		return name
	}
}

// createIndex creates an index for a column
func (sm *SchemaManager) createIndex(tableName, columnName string) error {
	indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
	quotedTable := sm.quoteIdentifier(tableName)
	quotedColumn := sm.quoteIdentifier(columnName)
	quotedIndex := sm.quoteIdentifier(indexName)

	query := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", quotedIndex, quotedTable, quotedColumn)
	
	_, err := sm.db.Exec(query)
	return err
}

// dropIndex drops an index for a column
func (sm *SchemaManager) dropIndex(tableName, columnName string) error {
	indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
	quotedIndex := sm.quoteIdentifier(indexName)

	var query string
	switch sm.dbType {
	case DatabaseTypeMySQL:
		quotedTable := sm.quoteIdentifier(tableName)
		query = fmt.Sprintf("DROP INDEX %s ON %s", quotedIndex, quotedTable)
	case DatabaseTypeSQLite:
		query = fmt.Sprintf("DROP INDEX IF EXISTS %s", quotedIndex)
	default:
		query = fmt.Sprintf("DROP INDEX %s", quotedIndex)
	}
	
	_, err := sm.db.Exec(query)
	return err
}

// migrateSchema migrates an existing table to match the schema
func (sm *SchemaManager) migrateSchema(schema *CollectionSchema) error {
	// Get current table structure
	currentColumns, err := sm.getTableColumns(schema.Name)
	if err != nil {
		return err
	}

	// Compare with desired schema and generate migrations
	migrations := sm.generateMigrations(currentColumns, schema.Columns)

	// Execute migrations
	for _, migration := range migrations {
		if err := sm.executeMigration(schema.Name, migration); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Type, err)
		}
	}

	return nil
}

// Migration represents a schema migration operation
type Migration struct {
	Type   string // ADD_COLUMN, DROP_COLUMN, MODIFY_COLUMN
	Column ColumnDefinition
	OldColumn *ColumnDefinition // For modify operations
}

// getTableColumns retrieves the current table column structure
func (sm *SchemaManager) getTableColumns(tableName string) ([]ColumnDefinition, error) {
	var columns []ColumnDefinition

	switch sm.dbType {
	case DatabaseTypeSQLite:
		query := fmt.Sprintf("PRAGMA table_info(%s)", sm.quoteIdentifier(tableName))
		rows, err := sm.db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull bool
			var dfltValue *string
			var pk bool

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				return nil, err
			}

			column := ColumnDefinition{
				Name:      name,
				Type:      ColumnType(strings.ToUpper(dataType)),
				Required:  notNull,
				IsPrimary: pk,
			}

			if dfltValue != nil {
				column.Default = *dfltValue
			}

			columns = append(columns, column)
		}

	case DatabaseTypeMySQL:
		query := `
			SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_KEY 
			FROM information_schema.COLUMNS 
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION
		`
		rows, err := sm.db.Query(query, tableName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var name, dataType, isNullable string
			var columnDefault *string
			var columnKey string

			if err := rows.Scan(&name, &dataType, &isNullable, &columnDefault, &columnKey); err != nil {
				return nil, err
			}

			column := ColumnDefinition{
				Name:      name,
				Type:      sm.mapMySQLTypeToColumn(dataType),
				Required:  isNullable == "NO",
				IsPrimary: columnKey == "PRI",
			}

			if columnDefault != nil {
				column.Default = *columnDefault
			}

			columns = append(columns, column)
		}
	}

	return columns, nil
}

// mapMySQLTypeToColumn maps MySQL data types to our column types
func (sm *SchemaManager) mapMySQLTypeToColumn(mysqlType string) ColumnType {
	switch strings.ToLower(mysqlType) {
	case "varchar", "text", "char":
		return ColumnTypeText
	case "int", "bigint", "smallint":
		return ColumnTypeInteger
	case "float", "double", "decimal":
		return ColumnTypeReal
	case "tinyint":
		return ColumnTypeBoolean
	case "datetime", "timestamp":
		return ColumnTypeDate
	case "json":
		return ColumnTypeJSON
	default:
		return ColumnTypeText
	}
}

// generateMigrations compares current and desired schemas to generate migrations
func (sm *SchemaManager) generateMigrations(current, desired []ColumnDefinition) []Migration {
	var migrations []Migration

	// Create maps for easier lookup
	currentMap := make(map[string]ColumnDefinition)
	for _, col := range current {
		currentMap[col.Name] = col
	}

	desiredMap := make(map[string]ColumnDefinition)
	for _, col := range desired {
		desiredMap[col.Name] = col
	}

	// Find columns to add (safe for all databases)
	for _, desiredCol := range desired {
		if _, exists := currentMap[desiredCol.Name]; !exists {
			migrations = append(migrations, Migration{
				Type:   "ADD_COLUMN",
				Column: desiredCol,
			})
		}
	}

	// Only attempt drops and modifications for databases that support them
	if sm.dbType != DatabaseTypeSQLite {
		// Find columns to drop (MySQL only - be careful here!)
		for _, currentCol := range current {
			if _, exists := desiredMap[currentCol.Name]; !exists && !sm.isSystemColumn(currentCol.Name) {
				migrations = append(migrations, Migration{
					Type:   "DROP_COLUMN",
					Column: currentCol,
				})
			}
		}

		// Find columns to modify (MySQL only)
		for _, desiredCol := range desired {
			if currentCol, exists := currentMap[desiredCol.Name]; exists {
				if sm.columnNeedsModification(currentCol, desiredCol) {
					migrations = append(migrations, Migration{
						Type:      "MODIFY_COLUMN",
						Column:    desiredCol,
						OldColumn: &currentCol,
					})
				}
			}
		}
	} else {
		// For SQLite, log warnings about unsupported changes instead of failing
		for _, currentCol := range current {
			if _, exists := desiredMap[currentCol.Name]; !exists && !sm.isSystemColumn(currentCol.Name) {
				fmt.Printf("Warning: Cannot drop column %s from SQLite table (not supported)\n", currentCol.Name)
			}
		}

		for _, desiredCol := range desired {
			if currentCol, exists := currentMap[desiredCol.Name]; exists {
				if sm.columnNeedsModification(currentCol, desiredCol) {
					fmt.Printf("Warning: Cannot modify column %s in SQLite table (not supported), keeping existing definition\n", desiredCol.Name)
				}
			}
		}
	}

	return migrations
}

// isSystemColumn checks if a column is a system column that shouldn't be dropped
func (sm *SchemaManager) isSystemColumn(columnName string) bool {
	systemColumns := []string{"id", "created_at", "updated_at", "data"}
	for _, sysCol := range systemColumns {
		if columnName == sysCol {
			return true
		}
	}
	return false
}

// columnNeedsModification checks if a column definition needs to be modified
func (sm *SchemaManager) columnNeedsModification(current, desired ColumnDefinition) bool {
	return current.Type != desired.Type ||
		   current.Required != desired.Required ||
		   !sm.defaultsEqual(current.Default, desired.Default)
}

// defaultsEqual compares two default values
func (sm *SchemaManager) defaultsEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// executeMigration executes a single migration
func (sm *SchemaManager) executeMigration(tableName string, migration Migration) error {
	quotedTable := sm.quoteIdentifier(tableName)
	quotedColumn := sm.quoteIdentifier(migration.Column.Name)

	switch migration.Type {
	case "ADD_COLUMN":
		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", quotedTable, sm.buildColumnDefinition(migration.Column))
		_, err := sm.db.Exec(query)
		if err != nil {
			return err
		}
		
		// Create index if requested
		if migration.Column.Index && !migration.Column.IsPrimary {
			if err := sm.createIndex(tableName, migration.Column.Name); err != nil {
				// Log warning but don't fail the migration
				fmt.Printf("Warning: failed to create index for %s.%s: %v\n", tableName, migration.Column.Name, err)
			}
		}
		return nil

	case "DROP_COLUMN":
		if sm.dbType == DatabaseTypeSQLite {
			// SQLite doesn't support DROP COLUMN directly, would need table recreation
			return fmt.Errorf("DROP COLUMN not supported for SQLite (would require table recreation)")
		}
		query := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quotedTable, quotedColumn)
		_, err := sm.db.Exec(query)
		return err

	case "MODIFY_COLUMN":
		if sm.dbType == DatabaseTypeSQLite {
			return fmt.Errorf("MODIFY COLUMN not supported for SQLite (would require table recreation)")
		}
		// MySQL
		query := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s", quotedTable, sm.buildColumnDefinition(migration.Column))
		_, err := sm.db.Exec(query)
		if err != nil {
			return err
		}
		
		// Handle index changes for modified columns
		if migration.Column.Index && !migration.Column.IsPrimary {
			// Check if index exists, if not create it
			if err := sm.createIndex(tableName, migration.Column.Name); err != nil {
				// Index might already exist, that's ok
				fmt.Printf("Info: index creation for %s.%s: %v\n", tableName, migration.Column.Name, err)
			}
		} else if migration.OldColumn != nil && migration.OldColumn.Index && !migration.Column.Index {
			// Remove index if it was removed from the schema
			if err := sm.dropIndex(tableName, migration.Column.Name); err != nil {
				fmt.Printf("Warning: failed to drop index for %s.%s: %v\n", tableName, migration.Column.Name, err)
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown migration type: %s", migration.Type)
	}
}