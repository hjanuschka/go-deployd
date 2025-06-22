package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	namespace  string
	collection *mongo.Collection
	db         *Database
}

func (s *Store) CreateUniqueIdentifier() string {
	bytes := make([]byte, 12)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *Store) Insert(ctx context.Context, document interface{}) (interface{}, error) {
	if doc, ok := document.(bson.M); ok {
		if _, exists := doc["id"]; !exists {
			doc["id"] = s.CreateUniqueIdentifier()
		}
		// Convert id to _id for MongoDB
		if id, exists := doc["id"]; exists {
			doc["_id"] = id
			delete(doc, "id")
		}
	}

	result, err := s.collection.InsertOne(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	// Get the inserted document to return it with the id field
	var inserted bson.M
	err = s.collection.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&inserted)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve inserted document: %w", err)
	}

	s.convertID(&inserted)
	return inserted, nil
}

func (s *Store) Find(ctx context.Context, query bson.M, opts ...*options.FindOptions) ([]bson.M, error) {
	s.scrubQuery(query)

	cursor, err := s.collection.Find(ctx, query, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode documents: %w", err)
	}

	// Convert _id back to id for all documents
	for i := range results {
		s.convertID(&results[i])
	}

	return results, nil
}

func (s *Store) FindOne(ctx context.Context, query bson.M) (bson.M, error) {
	s.scrubQuery(query)

	var result bson.M
	err := s.collection.FindOne(ctx, query).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	s.convertID(&result)
	return result, nil
}

func (s *Store) Update(ctx context.Context, query bson.M, update bson.M) (*mongo.UpdateResult, error) {
	s.scrubQuery(query)
	s.scrubUpdateOperations(update)

	result, err := s.collection.UpdateMany(ctx, query, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update documents: %w", err)
	}

	return result, nil
}

// UpdateOne updates a single document
func (s *Store) UpdateOne(ctx context.Context, query bson.M, update bson.M) (*mongo.UpdateResult, error) {
	s.scrubQuery(query)
	s.scrubUpdateOperations(update)

	result, err := s.collection.UpdateOne(ctx, query, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return result, nil
}

// Increment atomically increments numeric fields
func (s *Store) Increment(ctx context.Context, query bson.M, increments bson.M) (*mongo.UpdateResult, error) {
	update := bson.M{"$inc": increments}
	return s.Update(ctx, query, update)
}

// Push adds items to arrays
func (s *Store) Push(ctx context.Context, query bson.M, pushOps bson.M) (*mongo.UpdateResult, error) {
	update := bson.M{"$push": pushOps}
	return s.Update(ctx, query, update)
}

// Pull removes items from arrays
func (s *Store) Pull(ctx context.Context, query bson.M, pullOps bson.M) (*mongo.UpdateResult, error) {
	update := bson.M{"$pull": pullOps}
	return s.Update(ctx, query, update)
}

// AddToSet adds unique items to arrays (like $addUnique in original Deployd)
func (s *Store) AddToSet(ctx context.Context, query bson.M, addOps bson.M) (*mongo.UpdateResult, error) {
	update := bson.M{"$addToSet": addOps}
	return s.Update(ctx, query, update)
}

// PopFirst removes the first item from arrays
func (s *Store) PopFirst(ctx context.Context, query bson.M, fields []string) (*mongo.UpdateResult, error) {
	popOps := bson.M{}
	for _, field := range fields {
		popOps[field] = -1
	}
	update := bson.M{"$pop": popOps}
	return s.Update(ctx, query, update)
}

// PopLast removes the last item from arrays  
func (s *Store) PopLast(ctx context.Context, query bson.M, fields []string) (*mongo.UpdateResult, error) {
	popOps := bson.M{}
	for _, field := range fields {
		popOps[field] = 1
	}
	update := bson.M{"$pop": popOps}
	return s.Update(ctx, query, update)
}

// Upsert performs an update with upsert option
func (s *Store) Upsert(ctx context.Context, query bson.M, update bson.M) (*mongo.UpdateResult, error) {
	s.scrubQuery(query)
	s.scrubUpdateOperations(update)

	opts := options.Update().SetUpsert(true)
	result, err := s.collection.UpdateOne(ctx, query, update, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert document: %w", err)
	}

	return result, nil
}

// scrubUpdateOperations handles id/field conversion in update operations
func (s *Store) scrubUpdateOperations(update bson.M) {
	for op, value := range update {
		switch op {
		case "$set", "$unset", "$inc", "$push", "$pull", "$addToSet", "$pop", "$rename":
			if opDoc, ok := value.(bson.M); ok {
				s.scrubIDFields(opDoc)
			}
		case "$pushAll", "$pullAll":
			if opDoc, ok := value.(bson.M); ok {
				s.scrubIDFields(opDoc)
			}
		}
	}
}

// scrubIDFields converts id fields to _id in operation documents
func (s *Store) scrubIDFields(doc bson.M) {
	if _, hasID := doc["id"]; hasID {
		doc["_id"] = doc["id"]
		delete(doc, "id")
	}
}

func (s *Store) Remove(ctx context.Context, query bson.M) (*mongo.DeleteResult, error) {
	s.scrubQuery(query)

	result, err := s.collection.DeleteMany(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to delete documents: %w", err)
	}

	return result, nil
}

func (s *Store) Count(ctx context.Context, query bson.M) (int64, error) {
	s.scrubQuery(query)

	count, err := s.collection.CountDocuments(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

func (s *Store) Rename(ctx context.Context, newName string) error {
	// MongoDB doesn't have a direct rename collection method in the driver
	// This would need to be implemented using aggregation pipeline
	return fmt.Errorf("rename not yet implemented")
}

// scrubQuery converts "id" fields to "_id" for MongoDB compatibility
func (s *Store) scrubQuery(query bson.M) {
	if query == nil {
		return
	}

	if id, exists := query["id"]; exists {
		query["_id"] = id
		delete(query, "id")
	}

	// Handle nested queries (like $in, $or, etc.)
	for _, value := range query {
		switch v := value.(type) {
		case bson.M:
			s.scrubQuery(v)
		case []interface{}:
			for _, item := range v {
				if itemDoc, ok := item.(bson.M); ok {
					s.scrubQuery(itemDoc)
				}
			}
		}
	}
}

// convertID converts MongoDB's "_id" field back to "id" for API compatibility
func (s *Store) convertID(doc *bson.M) {
	if doc == nil {
		return
	}

	if id, exists := (*doc)["_id"]; exists {
		(*doc)["id"] = id
		delete(*doc, "_id")
	}
}

// Aggregate performs aggregation operations
func (s *Store) Aggregate(ctx context.Context, pipeline []bson.M) ([]bson.M, error) {
	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation results: %w", err)
	}

	// Convert _id back to id for all documents
	for i := range results {
		s.convertID(&results[i])
	}

	return results, nil
}