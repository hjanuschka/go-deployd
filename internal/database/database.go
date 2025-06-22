package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	Host string
	Port int
	Name string
}

type Database struct {
	client *mongo.Client
	db     *mongo.Database
	config *Config
}

func New(config *Config) (*Database, error) {
	uri := fmt.Sprintf("mongodb://%s:%d", config.Host, config.Port)
	
	clientOptions := options.Client().ApplyURI(uri)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	
	return &Database{
		client: client,
		db:     client.Database(config.Name),
		config: config,
	}, nil
}

func (d *Database) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.client.Disconnect(ctx)
}

func (d *Database) CreateStore(namespace string) *Store {
	return &Store{
		namespace:  namespace,
		collection: d.db.Collection(namespace),
		db:         d,
	}
}

func (d *Database) Drop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return d.db.Drop(ctx)
}