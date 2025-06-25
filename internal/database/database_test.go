package database

import (
	"os"
	"testing"
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
			name:   "Invalid database type",
			dbType: DatabaseType("invalid"),
			config: &Config{},
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