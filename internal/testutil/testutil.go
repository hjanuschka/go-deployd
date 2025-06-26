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
	"github.com/hjanuschka/go-deployd/internal/resources"
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

	var db database.DatabaseInterface
	var err error

	switch dbType {
	case "sqlite":
		dbPath := filepath.Join(t.TempDir(), "test.db")
		db, err = database.NewSQLiteDB(dbPath)
	case "mongodb":
		mongoURL := os.Getenv("TEST_MONGO_URL")
		if mongoURL == "" {
			mongoURL = "mongodb://localhost:27017/test_deployd"
		}
		db, err = database.NewMongoDB(mongoURL)
	case "mysql":
		mysqlDSN := os.Getenv("TEST_MYSQL_DSN")
		if mysqlDSN == "" {
			mysqlDSN = "root:password@tcp(localhost:3306)/test_deployd?parseTime=true"
		}
		db, err = database.NewMySQLDB(mysqlDSN)
	default:
		t.Fatalf("unsupported database type: %s", dbType)
	}

	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	return db
}

func CreateTestCollection(t *testing.T, db database.DatabaseInterface, collectionName string) *resources.Collection {
	if collectionName == "" {
		collectionName = GenerateRandomName("test_collection")
	}

	collection := &resources.Collection{
		Name: collectionName,
		Properties: map[string]resources.Property{
			"name": {
				Name:     "name",
				Type:     "string",
				Required: true,
			},
			"owner": {
				Name:     "owner",
				Type:     "string",
				Required: true,
			},
			"data": {
				Name:     "data",
				Type:     "object",
				Required: false,
			},
		},
	}

	store := db.GetStore(collectionName)
	if store == nil {
		t.Fatalf("failed to get store for collection %s", collectionName)
	}

	ctx := context.Background()
	err := store.CreateTable(ctx)
	if err != nil {
		t.Fatalf("failed to create table for collection %s: %v", collectionName, err)
	}

	return collection
}

func CleanupCollection(t *testing.T, db database.DatabaseInterface, collectionName string) {
	store := db.GetStore(collectionName)
	if store != nil {
		ctx := context.Background()
		_ = store.DropTable(ctx)
	}
}

type TestUser struct {
	ID       string
	Username string
	Email    string
	Token    string
}

func CreateTestUser(t *testing.T, db database.DatabaseInterface) *TestUser {
	userStore := db.GetStore("users")
	if userStore == nil {
		t.Fatal("failed to get users store")
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
	if id, ok := result.InsertedID.(string); ok {
		userID = id
	} else {
		t.Fatal("failed to get user ID")
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
		store := db.GetStore(collName)
		if store != nil {
			_ = store.DropTable(ctx)
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