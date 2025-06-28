package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (r *MongoUpdateResult) ModifiedCount() int64 { return r.UpdateResult.ModifiedCount }
func (r *MongoUpdateResult) UpsertedCount() int64 {
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
	legacyConfig := &LegacyConfig{
		Host: config.Host,
		Port: config.Port,
		Name: config.Name,
	}
	db, err := New(legacyConfig)
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

	// Convert []bson.M to []map[string]interface{} with BSON conversion
	mapResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		mapResults[i] = s.convertBSONToMap(map[string]interface{}(result))
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

	// Convert bson.M to map[string]interface{} recursively
	return s.convertBSONToMap(map[string]interface{}(result)), nil
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

// Enhanced MongoDB-style query methods
func (s *MongoStore) FindWithRawQuery(ctx context.Context, mongoQuery interface{}, queryOptions map[string]interface{}) ([]map[string]interface{}, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	// Convert to BSON
	bsonQuery := s.mapToBSON(parsedQuery)

	// Build find options
	findOpts := options.Find()
	if sort, exists := queryOptions["$sort"]; exists {
		if sortMap, ok := sort.(map[string]interface{}); ok {
			sortBSON := bson.D{}
			for field, direction := range sortMap {
				if dir, ok := direction.(int); ok {
					sortBSON = append(sortBSON, bson.E{Key: field, Value: dir})
				}
			}
			findOpts.SetSort(sortBSON)
		}
	}

	if limit, exists := queryOptions["$limit"]; exists {
		if limitInt, ok := limit.(int); ok {
			findOpts.SetLimit(int64(limitInt))
		}
	}

	if skip, exists := queryOptions["$skip"]; exists {
		if skipInt, ok := skip.(int); ok {
			findOpts.SetSkip(int64(skipInt))
		}
	}

	if fields, exists := queryOptions["$fields"]; exists {
		if fieldsMap, ok := fields.(map[string]interface{}); ok {
			projection := bson.M{}
			for field, include := range fieldsMap {
				projection[field] = include
			}
			findOpts.SetProjection(projection)
		}
	}

	// Execute query
	results, err := s.Store.Find(ctx, bsonQuery, findOpts)
	if err != nil {
		return nil, err
	}

	// Convert results
	mapResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		mapResults[i] = s.convertBSONToMap(map[string]interface{}(result))
	}

	return mapResults, nil
}

func (s *MongoStore) CountWithRawQuery(ctx context.Context, mongoQuery interface{}) (int64, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return 0, err
	}

	// Convert to BSON and count
	bsonQuery := s.mapToBSON(parsedQuery)
	return s.Store.Count(ctx, bsonQuery)
}

func (s *MongoStore) UpdateWithRawQuery(ctx context.Context, mongoQuery interface{}, mongoUpdate interface{}) (UpdateResult, error) {
	// Parse the MongoDB query and update
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	parsedUpdate, err := ParseMongoQuery(mongoUpdate)
	if err != nil {
		return nil, err
	}

	// Convert to BSON
	bsonQuery := s.mapToBSON(parsedQuery)
	bsonUpdate := s.mapToBSON(parsedUpdate)

	// Execute update
	result, err := s.Store.Update(ctx, bsonQuery, bsonUpdate)
	if err != nil {
		return nil, err
	}

	return &MongoUpdateResult{UpdateResult: result}, nil
}

func (s *MongoStore) RemoveWithRawQuery(ctx context.Context, mongoQuery interface{}) (DeleteResult, error) {
	// Parse the MongoDB query
	parsedQuery, err := ParseMongoQuery(mongoQuery)
	if err != nil {
		return nil, err
	}

	// Convert to BSON and remove
	bsonQuery := s.mapToBSON(parsedQuery)
	result, err := s.Store.Remove(ctx, bsonQuery)
	if err != nil {
		return nil, err
	}

	return &MongoDeleteResult{DeleteResult: result}, nil
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

// convertBSONToMap recursively converts BSON types to standard Go types
func (s *MongoStore) convertBSONToMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		result[key] = s.convertBSONValue(value)
	}

	return result
}

// convertBSONValue converts individual BSON values to standard Go types
func (s *MongoStore) convertBSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		// Recursively convert nested maps
		return s.convertBSONToMap(v)
	case primitive.M:
		// Convert primitive.M (which is the same as bson.M) to map[string]interface{}
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = s.convertBSONValue(val)
		}
		return result
	case primitive.A:
		// Convert primitive.A (array) to []interface{}
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = s.convertBSONValue(item)
		}
		return result
	case []interface{}:
		// Convert arrays recursively
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = s.convertBSONValue(item)
		}
		return result
	default:
		// For primitive types, return as-is
		return value
	}
}

// Register MongoDB database factory
func init() {
	RegisterDatabaseFactory(DatabaseTypeMongoDB, NewMongoDatabase)
}
