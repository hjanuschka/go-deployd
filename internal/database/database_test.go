package database

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDatabase(t *testing.T) {
	tests := []struct {
		name    string
		dbType  DatabaseType
		config  *Config
		wantErr bool
	}{
		{
			name:   "SQLite database",
			dbType: DatabaseTypeSQLite,
			config: &Config{
				Name: ":memory:",
			},
			wantErr: false,
		},
		{
			name:    "Invalid database type",
			dbType:  DatabaseType("invalid"),
			config:  &Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDatabase(tt.dbType, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDatabase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if db != nil {
				db.Close()
			}
		})
	}
}

func TestGenerateUniqueID(t *testing.T) {
	// Test that generateUniqueID returns unique values
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateUniqueID()
		if ids[id] {
			t.Errorf("generateUniqueID() returned duplicate ID: %s", id)
		}
		ids[id] = true

		// Check ID format (should be hex string)
		if len(id) != 24 {
			t.Errorf("generateUniqueID() returned ID with wrong length: %d (expected 24)", len(id))
		}
	}
}

// Helper function to create a test SQLite database
func createTestSQLiteDB(t *testing.T) DatabaseInterface {
	db, err := NewDatabase(DatabaseTypeSQLite, &Config{Name: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

// Cleanup function to remove test databases
func cleanupTestDB(db DatabaseInterface) {
	if db != nil {
		db.Close()
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}

func TestDatabaseOperations(t *testing.T) {
	t.Run("Create multiple stores", func(t *testing.T) {
		db := createTestSQLiteDB(t)
		defer cleanupTestDB(db)

		// Create multiple stores
		store1 := db.CreateStore("collection1")
		store2 := db.CreateStore("collection2")

		assert.NotNil(t, store1)
		assert.NotNil(t, store2)

		// Test unique identifiers
		id1 := store1.CreateUniqueIdentifier()
		id2 := store2.CreateUniqueIdentifier()

		assert.NotEqual(t, id1, id2)
		assert.Len(t, id1, 24)
		assert.Len(t, id2, 24)

		t.Log("✅ Multiple stores created successfully")
	})

	t.Run("Database close operation", func(t *testing.T) {
		db := createTestSQLiteDB(t)

		// Test that close doesn't panic
		db.Close()

		t.Log("✅ Database close operation works")
	})
}

func TestSQLiteStoreOperations(t *testing.T) {
	t.Run("Insert and retrieve operations", func(t *testing.T) {
		db := createTestSQLiteDB(t)
		defer cleanupTestDB(db)

		store := db.CreateStore("test_ops")
		ctx := context.Background()

		// Test different data types
		testCases := []map[string]interface{}{
			{
				"name":   "String Test",
				"age":    25,
				"active": true,
				"score":  95.5,
			},
			{
				"name":     "Complex Test",
				"metadata": map[string]interface{}{"key": "value"},
				"tags":     []string{"tag1", "tag2"},
			},
		}

		for i, testDoc := range testCases {
			inserted, err := store.Insert(ctx, testDoc)
			assert.NoError(t, err, "Insert %d should succeed", i)
			assert.NotNil(t, inserted, "Insert %d should return data", i)
		}

		t.Log("✅ Insert operations work with different data types")
	})

	t.Run("Count with empty query", func(t *testing.T) {
		db := createTestSQLiteDB(t)
		defer cleanupTestDB(db)

		store := db.CreateStore("count_test")
		ctx := context.Background()

		// Insert some test data
		for i := 0; i < 3; i++ {
			doc := map[string]interface{}{
				"item": i,
				"name": fmt.Sprintf("item_%d", i),
			}
			_, err := store.Insert(ctx, doc)
			assert.NoError(t, err)
		}

		// Test count
		query := &SimpleQueryBuilder{conditions: make(map[string]interface{})}
		count, err := store.Count(ctx, query)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))

		t.Log("✅ Count operations work correctly")
	})
}
