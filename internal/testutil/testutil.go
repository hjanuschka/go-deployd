package testutil

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hjanuschka/go-deployd/internal/database"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateRandomName(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String()[:8])
}

func CreateTestDB(t *testing.T) database.DatabaseInterface {
	dbType := os.Getenv("TEST_DB")
	if dbType == "" {
		dbType = "sqlite"
	}

	var config *database.Config
	var err error

	switch dbType {
	case "sqlite":
		dbPath := filepath.Join(t.TempDir(), "test.db")
		config = &database.Config{
			Name: dbPath,
		}
	case "mongodb":
		mongoURL := os.Getenv("TEST_MONGO_URL")
		if mongoURL == "" {
			mongoURL = "mongodb://localhost:27017"
		}
		config = &database.Config{
			Host: "localhost",
			Port: 27017,
			Name: "test_deployd_" + GenerateRandomName("db"),
		}
	case "mysql":
		config = &database.Config{
			Host:     os.Getenv("TEST_MYSQL_HOST"),
			Port:     3306,
			Name:     "test_deployd_" + GenerateRandomName("db"),
			Username: os.Getenv("TEST_MYSQL_USER"),
			Password: os.Getenv("TEST_MYSQL_PASSWORD"),
		}
		if config.Host == "" {
			config.Host = "localhost"
		}
		if config.Username == "" {
			config.Username = "root"
		}
		if config.Password == "" {
			config.Password = "password"
		}
	default:
		t.Fatalf("unsupported database type: %s", dbType)
	}

	db, err := database.NewDatabase(database.DatabaseType(dbType), config)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	return db
}

func CreateTestCollection(t *testing.T, db database.DatabaseInterface, collectionName string) database.StoreInterface {
	if collectionName == "" {
		collectionName = GenerateRandomName("test_collection")
	}

	store := db.CreateStore(collectionName)
	if store == nil {
		t.Fatalf("failed to create store for collection %s", collectionName)
	}

	return store
}

func CleanupCollection(t *testing.T, db database.DatabaseInterface, collectionName string) {
	store := db.CreateStore(collectionName)
	if store != nil {
		ctx := context.Background()
		// Try to remove all documents from the collection
		emptyQuery := database.NewQueryBuilder()
		_, _ = store.Remove(ctx, emptyQuery)
	}
}

type TestUser struct {
	ID       string
	Username string
	Email    string
	Token    string
}

func CreateTestUser(t *testing.T, db database.DatabaseInterface) *TestUser {
	userStore := db.CreateStore("users")
	if userStore == nil {
		t.Fatal("failed to create users store")
	}

	username := GenerateRandomName("testuser")
	email := fmt.Sprintf("%s@test.com", username)
	
	ctx := context.Background()
	userData := map[string]interface{}{
		"username": username,
		"email":    email,
		"password": "hashedpassword123",
		"verified": true,
	}

	result, err := userStore.Insert(ctx, userData)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	userID := ""
	if id, ok := result.(string); ok {
		userID = id
	} else if insertResult, ok := result.(map[string]interface{}); ok {
		if id, exists := insertResult["InsertedID"]; exists {
			userID = id.(string)
		}
	} else {
		userID = userStore.CreateUniqueIdentifier()
	}

	return &TestUser{
		ID:       userID,
		Username: username,
		Email:    email,
	}
}

func CleanupTestData(t *testing.T, db database.DatabaseInterface, collections []string) {
	ctx := context.Background()
	for _, collName := range collections {
		store := db.CreateStore(collName)
		if store != nil {
			emptyQuery := database.NewQueryBuilder()
			_, _ = store.Remove(ctx, emptyQuery)
		}
	}
}

type TestContext struct {
	UserID   string
	Username string
	IsAdmin  bool
	Extra    map[string]interface{}
}

func (tc *TestContext) Value(key interface{}) interface{} {
	switch key.(type) {
	case string:
		switch key {
		case "userID":
			return tc.UserID
		case "username":
			return tc.Username
		case "isAdmin":
			return tc.IsAdmin
		default:
			if tc.Extra != nil {
				return tc.Extra[key.(string)]
			}
		}
	}
	return nil
}

func CreateTestContext(userID, username string, isAdmin bool) context.Context {
	tc := &TestContext{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		Extra:    make(map[string]interface{}),
	}
	return context.WithValue(context.Background(), "testContext", tc)
}