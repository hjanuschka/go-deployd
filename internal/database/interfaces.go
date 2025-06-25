package database

import (
	"context"
)

// DatabaseType represents the type of database backend
type DatabaseType string

const (
	DatabaseTypeMongoDB DatabaseType = "mongodb"
	DatabaseTypeSQLite  DatabaseType = "sqlite"
	DatabaseTypeMySQL   DatabaseType = "mysql"
	DatabaseTypePostgres DatabaseType = "postgres"
)

// Config represents database connection configuration
type Config struct {
	Host     string
	Port     int
	Name     string
	Username string // For MySQL/PostgreSQL
	Password string // For MySQL/PostgreSQL
	SSL      bool   // For MySQL/PostgreSQL
}

// DatabaseInterface defines the common interface for all database implementations
type DatabaseInterface interface {
	CreateStore(namespace string) StoreInterface
	Close() error
	Drop() error
	GetType() DatabaseType
}

// StoreInterface defines the common interface for all store implementations
type StoreInterface interface {
	CreateUniqueIdentifier() string
	Insert(ctx context.Context, document interface{}) (interface{}, error)
	Find(ctx context.Context, query QueryBuilder, opts QueryOptions) ([]map[string]interface{}, error)
	FindOne(ctx context.Context, query QueryBuilder) (map[string]interface{}, error)
	Update(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error)
	UpdateOne(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error)
	Remove(ctx context.Context, query QueryBuilder) (DeleteResult, error)
	Count(ctx context.Context, query QueryBuilder) (int64, error)
	
	// MongoDB-style operations that need translation for SQL
	Increment(ctx context.Context, query QueryBuilder, increments map[string]interface{}) (UpdateResult, error)
	Push(ctx context.Context, query QueryBuilder, pushOps map[string]interface{}) (UpdateResult, error)
	Pull(ctx context.Context, query QueryBuilder, pullOps map[string]interface{}) (UpdateResult, error)
	AddToSet(ctx context.Context, query QueryBuilder, addOps map[string]interface{}) (UpdateResult, error)
	PopFirst(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error)
	PopLast(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error)
	Upsert(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error)
	Aggregate(ctx context.Context, pipeline []map[string]interface{}) ([]map[string]interface{}, error)
}

// QueryBuilder provides a database-agnostic query building interface
type QueryBuilder interface {
	Where(field string, operator string, value interface{}) QueryBuilder
	WhereIn(field string, values []interface{}) QueryBuilder
	WhereNotIn(field string, values []interface{}) QueryBuilder
	WhereNull(field string) QueryBuilder
	WhereNotNull(field string) QueryBuilder
	WhereRegex(field string, pattern string) QueryBuilder
	Or(conditions ...QueryBuilder) QueryBuilder
	And(conditions ...QueryBuilder) QueryBuilder
	Clone() QueryBuilder
	ToMap() map[string]interface{} // For MongoDB compatibility
}

// QueryOptions represents query options like sorting, limiting, etc.
type QueryOptions struct {
	Sort   map[string]int // field -> direction (1 or -1)
	Limit  *int64
	Skip   *int64
	Fields map[string]int // field -> include (1) or exclude (0)
}

// UpdateBuilder provides a database-agnostic update building interface
type UpdateBuilder interface {
	Set(field string, value interface{}) UpdateBuilder
	Unset(field string) UpdateBuilder
	Inc(field string, value interface{}) UpdateBuilder
	Push(field string, value interface{}) UpdateBuilder
	Pull(field string, value interface{}) UpdateBuilder
	AddToSet(field string, value interface{}) UpdateBuilder
	ToMap() map[string]interface{} // For MongoDB compatibility
}

// UpdateResult represents the result of an update operation
type UpdateResult interface {
	ModifiedCount() int64
	UpsertedCount() int64
	UpsertedID() interface{}
}

// DeleteResult represents the result of a delete operation
type DeleteResult interface {
	DeletedCount() int64
}

// Factory function type for creating database instances
type DatabaseFactory func(config *Config) (DatabaseInterface, error)

// Registry for database factories
var databaseFactories = make(map[DatabaseType]DatabaseFactory)

// RegisterDatabaseFactory registers a new database factory
func RegisterDatabaseFactory(dbType DatabaseType, factory DatabaseFactory) {
	databaseFactories[dbType] = factory
}

// NewDatabase creates a new database instance based on the type
func NewDatabase(dbType DatabaseType, config *Config) (DatabaseInterface, error) {
	factory, exists := databaseFactories[dbType]
	if !exists {
		return nil, &UnsupportedDatabaseError{Type: dbType}
	}
	return factory(config)
}

// UnsupportedDatabaseError is returned when an unsupported database type is requested
type UnsupportedDatabaseError struct {
	Type DatabaseType
}

func (e *UnsupportedDatabaseError) Error() string {
	return "unsupported database type: " + string(e.Type)
}