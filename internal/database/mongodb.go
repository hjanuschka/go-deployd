package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDatabase wraps the existing Database struct to implement DatabaseInterface
type MongoDatabase struct {
	*Database
}

// MongoStore wraps the existing Store struct to implement StoreInterface
type MongoStore struct {
	*Store
}

// MongoUpdateResult wraps mongo.UpdateResult to implement UpdateResult interface
type MongoUpdateResult struct {
	*mongo.UpdateResult
}

func (r *MongoUpdateResult) ModifiedCount() int64   { return r.UpdateResult.ModifiedCount }
func (r *MongoUpdateResult) UpsertedCount() int64   { 
	if r.UpdateResult.UpsertedID != nil {
		return 1
	}
	return 0
}
func (r *MongoUpdateResult) UpsertedID() interface{} { return r.UpdateResult.UpsertedID }

// MongoDeleteResult wraps mongo.DeleteResult to implement DeleteResult interface
type MongoDeleteResult struct {
	*mongo.DeleteResult
}

func (r *MongoDeleteResult) DeletedCount() int64 { return r.DeleteResult.DeletedCount }

// NewMongoDatabase creates a new MongoDB database instance that implements DatabaseInterface
func NewMongoDatabase(config *Config) (DatabaseInterface, error) {
	db, err := New(config)
	if err != nil {
		return nil, err
	}
	return &MongoDatabase{Database: db}, nil
}

func (d *MongoDatabase) CreateStore(namespace string) StoreInterface {
	store := d.Database.CreateStore(namespace)
	return &MongoStore{Store: store}
}

func (d *MongoDatabase) GetType() DatabaseType {
	return DatabaseTypeMongoDB
}

// Adapter methods for MongoStore to implement StoreInterface

func (s *MongoStore) Find(ctx context.Context, query QueryBuilder, opts QueryOptions) ([]map[string]interface{}, error) {
	// Convert QueryBuilder to bson.M
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)

	// Convert QueryOptions to MongoDB options
	findOpts := options.Find()
	
	if len(opts.Sort) > 0 {
		sortBSON := bson.D{}
		for field, direction := range opts.Sort {
			sortBSON = append(sortBSON, bson.E{Key: field, Value: direction})
		}
		findOpts.SetSort(sortBSON)
	}
	
	if opts.Limit != nil {
		findOpts.SetLimit(*opts.Limit)
	}
	
	if opts.Skip != nil {
		findOpts.SetSkip(*opts.Skip)
	}
	
	if len(opts.Fields) > 0 {
		projection := bson.M{}
		for field, include := range opts.Fields {
			projection[field] = include
		}
		findOpts.SetProjection(projection)
	}

	// Use the existing Find method
	results, err := s.Store.Find(ctx, bsonQuery, findOpts)
	if err != nil {
		return nil, err
	}

	// Convert []bson.M to []map[string]interface{}
	mapResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		mapResults[i] = map[string]interface{}(result)
	}

	return mapResults, nil
}

func (s *MongoStore) FindOne(ctx context.Context, query QueryBuilder) (map[string]interface{}, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	
	result, err := s.Store.FindOne(ctx, bsonQuery)
	if err != nil {
		return nil, err
	}
	
	if result == nil {
		return nil, nil
	}
	
	return map[string]interface{}(result), nil
}

func (s *MongoStore) Update(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	queryMap := query.ToMap()
	updateMap := update.ToMap()
	
	bsonQuery := s.mapToBSON(queryMap)
	bsonUpdate := s.mapToBSON(updateMap)
	
	result, err := s.Store.Update(ctx, bsonQuery, bsonUpdate)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) UpdateOne(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	queryMap := query.ToMap()
	updateMap := update.ToMap()
	
	bsonQuery := s.mapToBSON(queryMap)
	bsonUpdate := s.mapToBSON(updateMap)
	
	result, err := s.Store.UpdateOne(ctx, bsonQuery, bsonUpdate)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) Remove(ctx context.Context, query QueryBuilder) (DeleteResult, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	
	result, err := s.Store.Remove(ctx, bsonQuery)
	if err != nil {
		return nil, err
	}
	
	return &MongoDeleteResult{DeleteResult: result}, nil
}

func (s *MongoStore) Count(ctx context.Context, query QueryBuilder) (int64, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	
	return s.Store.Count(ctx, bsonQuery)
}

func (s *MongoStore) Increment(ctx context.Context, query QueryBuilder, increments map[string]interface{}) (UpdateResult, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	bsonInc := s.mapToBSON(increments)
	
	result, err := s.Store.Increment(ctx, bsonQuery, bsonInc)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) Push(ctx context.Context, query QueryBuilder, pushOps map[string]interface{}) (UpdateResult, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	bsonPush := s.mapToBSON(pushOps)
	
	result, err := s.Store.Push(ctx, bsonQuery, bsonPush)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) Pull(ctx context.Context, query QueryBuilder, pullOps map[string]interface{}) (UpdateResult, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	bsonPull := s.mapToBSON(pullOps)
	
	result, err := s.Store.Pull(ctx, bsonQuery, bsonPull)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) AddToSet(ctx context.Context, query QueryBuilder, addOps map[string]interface{}) (UpdateResult, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	bsonAdd := s.mapToBSON(addOps)
	
	result, err := s.Store.AddToSet(ctx, bsonQuery, bsonAdd)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) PopFirst(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	
	result, err := s.Store.PopFirst(ctx, bsonQuery, fields)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) PopLast(ctx context.Context, query QueryBuilder, fields []string) (UpdateResult, error) {
	queryMap := query.ToMap()
	bsonQuery := s.mapToBSON(queryMap)
	
	result, err := s.Store.PopLast(ctx, bsonQuery, fields)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) Upsert(ctx context.Context, query QueryBuilder, update UpdateBuilder) (UpdateResult, error) {
	queryMap := query.ToMap()
	updateMap := update.ToMap()
	
	bsonQuery := s.mapToBSON(queryMap)
	bsonUpdate := s.mapToBSON(updateMap)
	
	result, err := s.Store.Upsert(ctx, bsonQuery, bsonUpdate)
	if err != nil {
		return nil, err
	}
	
	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) Aggregate(ctx context.Context, pipeline []map[string]interface{}) ([]map[string]interface{}, error) {
	// Convert []map[string]interface{} to []bson.M
	bsonPipeline := make([]bson.M, len(pipeline))
	for i, stage := range pipeline {
		bsonPipeline[i] = s.mapToBSON(stage)
	}
	
	results, err := s.Store.Aggregate(ctx, bsonPipeline)
	if err != nil {
		return nil, err
	}
	
	// Convert []bson.M to []map[string]interface{}
	mapResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		mapResults[i] = map[string]interface{}(result)
	}
	
	return mapResults, nil
}

// Helper method to convert map[string]interface{} to bson.M
func (s *MongoStore) mapToBSON(m map[string]interface{}) bson.M {
	if m == nil {
		return nil
	}
	
	result := make(bson.M)
	for k, v := range m {
		result[k] = v
	}
	return result
}

// Register MongoDB database factory
func init() {
	RegisterDatabaseFactory(DatabaseTypeMongoDB, NewMongoDatabase)
}