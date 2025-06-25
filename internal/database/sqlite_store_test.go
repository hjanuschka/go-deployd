package database

import (
	"context"
	"testing"
)

func TestSQLiteStore_CreateStore(t *testing.T) {
	db := createTestSQLiteDB(t)
	defer cleanupTestDB(db)

	store := db.CreateStore("test_collection")
	if store == nil {
		t.Fatal("Expected store to be created")
	}

	// Test that we can create a unique identifier
	id := store.CreateUniqueIdentifier()
	if id == "" {
		t.Error("Expected non-empty unique identifier")
	}

	if len(id) != 24 {
		t.Errorf("Expected ID length 24, got %d", len(id))
	}
}

func TestSQLiteStore_Insert(t *testing.T) {
	db := createTestSQLiteDB(t)
	defer cleanupTestDB(db)

	store := db.CreateStore("test_collection")
	ctx := context.Background()

	testDoc := map[string]interface{}{
		"name":   "Test User",
		"email":  "test@example.com",
		"age":    30,
		"active": true,
	}

	// Insert document
	inserted, err := store.Insert(ctx, testDoc)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Check that something was returned
	if inserted == nil {
		t.Errorf("Expected non-nil result from insert")
	}
}

func TestSQLiteStore_Count(t *testing.T) {
	db := createTestSQLiteDB(t)
	defer cleanupTestDB(db)

	store := db.CreateStore("test_collection")
	ctx := context.Background()

	// Insert test documents
	for i := 0; i < 5; i++ {
		doc := map[string]interface{}{
			"name":   "User",
			"number": i,
		}
		_, err := store.Insert(ctx, doc)
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	// Create a simple query builder (we'll need to implement this based on the actual interface)
	query := &SimpleQueryBuilder{conditions: make(map[string]interface{})}

	count, err := store.Count(ctx, query)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected count=5, got %d", count)
	}
}

// SimpleQueryBuilder is a basic implementation for testing
type SimpleQueryBuilder struct {
	conditions map[string]interface{}
}

func (q *SimpleQueryBuilder) Where(field string, operator string, value interface{}) QueryBuilder {
	q.conditions[field] = value
	return q
}

func (q *SimpleQueryBuilder) WhereIn(field string, values []interface{}) QueryBuilder {
	q.conditions[field] = map[string]interface{}{"$in": values}
	return q
}

func (q *SimpleQueryBuilder) WhereNotIn(field string, values []interface{}) QueryBuilder {
	q.conditions[field] = map[string]interface{}{"$nin": values}
	return q
}

func (q *SimpleQueryBuilder) WhereNull(field string) QueryBuilder {
	q.conditions[field] = nil
	return q
}

func (q *SimpleQueryBuilder) WhereNotNull(field string) QueryBuilder {
	q.conditions[field] = map[string]interface{}{"$ne": nil}
	return q
}

func (q *SimpleQueryBuilder) WhereRegex(field string, pattern string) QueryBuilder {
	q.conditions[field] = map[string]interface{}{"$regex": pattern}
	return q
}

func (q *SimpleQueryBuilder) Or(conditions ...QueryBuilder) QueryBuilder {
	// Simple implementation
	return q
}

func (q *SimpleQueryBuilder) And(conditions ...QueryBuilder) QueryBuilder {
	// Simple implementation
	return q
}

func (q *SimpleQueryBuilder) Clone() QueryBuilder {
	clone := &SimpleQueryBuilder{conditions: make(map[string]interface{})}
	for k, v := range q.conditions {
		clone.conditions[k] = v
	}
	return clone
}

func (q *SimpleQueryBuilder) ToMap() map[string]interface{} {
	return q.conditions
}