package events

import (
	"fmt"
	"github.com/hjanuschka/go-deployd/internal/resources"
	"go.mongodb.org/mongo-driver/bson"
)

// InternalStore provides internal access to collection data
type InternalStore struct {
	collection *resources.Collection
}

// NewInternalStore creates a new internal store for a collection
func NewInternalStore(collection *resources.Collection) *InternalStore {
	return &InternalStore{
		collection: collection,
	}
}

// Find retrieves multiple documents based on query
func (s *InternalStore) Find(query bson.M, options ...bson.M) ([]bson.M, error) {
	if s.collection == nil {
		return nil, fmt.Errorf("collection not found")
	}
	
	// Apply options if provided
	opts := bson.M{}
	if len(options) > 0 {
		opts = options[0]
	}
	
	// Merge query with options
	fullQuery := bson.M{}
	for k, v := range query {
		fullQuery[k] = v
	}
	for k, v := range opts {
		fullQuery[k] = v
	}
	
	// Use collection's internal Get method
	result, err := s.collection.Get("", fullQuery)
	if err != nil {
		return nil, err
	}
	
	// Convert result to []bson.M
	if docs, ok := result.([]bson.M); ok {
		return docs, nil
	}
	
	// Try single document
	if doc, ok := result.(bson.M); ok {
		return []bson.M{doc}, nil
	}
	
	return nil, fmt.Errorf("unexpected result type")
}

// FindOne retrieves a single document based on query
func (s *InternalStore) FindOne(query bson.M) (bson.M, error) {
	docs, err := s.Find(query, bson.M{"$limit": 1})
	if err != nil {
		return nil, err
	}
	
	if len(docs) > 0 {
		return docs[0], nil
	}
	
	return nil, fmt.Errorf("document not found")
}

// Insert creates a new document
func (s *InternalStore) Insert(data bson.M) (bson.M, error) {
	if s.collection == nil {
		return nil, fmt.Errorf("collection not found")
	}
	
	result, err := s.collection.Create(data)
	if err != nil {
		return nil, err
	}
	
	if doc, ok := result.(bson.M); ok {
		return doc, nil
	}
	
	return nil, fmt.Errorf("unexpected result type")
}

// Update modifies an existing document
func (s *InternalStore) Update(id string, data bson.M) (bson.M, error) {
	if s.collection == nil {
		return nil, fmt.Errorf("collection not found")
	}
	
	result, err := s.collection.Update(id, data)
	if err != nil {
		return nil, err
	}
	
	if doc, ok := result.(bson.M); ok {
		return doc, nil
	}
	
	return nil, fmt.Errorf("unexpected result type")
}

// Delete removes a document
func (s *InternalStore) Delete(id string) error {
	if s.collection == nil {
		return fmt.Errorf("collection not found")
	}
	
	_, err := s.collection.Delete(id)
	return err
}

// Count returns the number of documents matching the query
func (s *InternalStore) Count(query bson.M) (int64, error) {
	docs, err := s.Find(query)
	if err != nil {
		return 0, err
	}
	
	return int64(len(docs)), nil
}

// InternalAPI provides access to all collections
type InternalAPI struct {
	router interface {
		GetCollection(name string) *resources.Collection
	}
}

// NewInternalAPI creates a new internal API instance
func NewInternalAPI(router interface {
	GetCollection(name string) *resources.Collection
}) *InternalAPI {
	return &InternalAPI{
		router: router,
	}
}

// Collection returns an internal store for the named collection
func (api *InternalAPI) Collection(name string) *InternalStore {
	if api.router == nil {
		return &InternalStore{collection: nil}
	}
	
	collection := api.router.GetCollection(name)
	return NewInternalStore(collection)
}