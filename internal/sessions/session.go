package sessions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"github.com/hjanuschka/go-deployd/internal/database"
)

type SessionStore struct {
	store       *database.Store
	development bool
}

type Session struct {
	ID          string                 `bson:"id" json:"id"`
	Data        map[string]interface{} `bson:"data" json:"data"`
	CreatedAt   time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `bson:"updatedAt" json:"updatedAt"`
	development bool
}

func New(db *database.Database, development bool) *SessionStore {
	return &SessionStore{
		store:       db.CreateStore("sessions"),
		development: development,
	}
}

func (ss *SessionStore) CreateSession(sessionID string) (*Session, error) {
	ctx := context.Background()
	
	if sessionID != "" {
		// Try to find existing session
		existing, err := ss.store.FindOne(ctx, bson.M{"id": sessionID})
		if err == nil && existing != nil {
			session := &Session{
				ID:          sessionID,
				Data:        make(map[string]interface{}),
				development: ss.development,
			}
			
			if data, exists := existing["data"]; exists {
				if dataMap, ok := data.(map[string]interface{}); ok {
					session.Data = dataMap
				}
			}
			
			if createdAt, exists := existing["createdAt"]; exists {
				if t, ok := createdAt.(time.Time); ok {
					session.CreatedAt = t
				}
			}
			
			if updatedAt, exists := existing["updatedAt"]; exists {
				if t, ok := updatedAt.(time.Time); ok {
					session.UpdatedAt = t
				}
			}
			
			return session, nil
		}
	}
	
	// Create new session
	newSessionID := ss.generateSessionID()
	now := time.Now()
	
	session := &Session{
		ID:          newSessionID,
		Data:        make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		development: ss.development,
	}
	
	// Save to database
	sessionDoc := bson.M{
		"id":        session.ID,
		"data":      session.Data,
		"createdAt": session.CreatedAt,
		"updatedAt": session.UpdatedAt,
	}
	
	_, err := ss.store.Insert(ctx, sessionDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	return session, nil
}

func (ss *SessionStore) generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *Session) GetID() string {
	return s.ID
}

func (s *Session) IsRoot() bool {
	if s.development {
		return true // In development mode, all sessions are root
	}
	
	if root, exists := s.Data["isRoot"]; exists {
		if rootBool, ok := root.(bool); ok {
			return rootBool
		}
	}
	
	return false
}

func (s *Session) Get(key string) interface{} {
	return s.Data[key]
}

func (s *Session) Set(key string, value interface{}) {
	s.Data[key] = value
	s.UpdatedAt = time.Now()
}

func (s *Session) Save(store *SessionStore) error {
	ctx := context.Background()
	
	update := bson.M{
		"$set": bson.M{
			"data":      s.Data,
			"updatedAt": s.UpdatedAt,
		},
	}
	
	_, err := store.store.Update(ctx, bson.M{"id": s.ID}, update)
	return err
}

func (ss *SessionStore) GetSessionFromRequest(r *http.Request) (*Session, error) {
	// Try to get session ID from cookie
	cookie, err := r.Cookie("sid")
	if err == nil && cookie.Value != "" {
		return ss.CreateSession(cookie.Value)
	}
	
	// Try to get session ID from Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" && len(auth) > 7 && auth[:7] == "Bearer " {
		token := auth[7:]
		return ss.CreateSession(token)
	}
	
	// Create new session
	return ss.CreateSession("")
}

func (ss *SessionStore) SetSessionCookie(w http.ResponseWriter, session *Session) {
	cookie := &http.Cookie{
		Name:     "sid",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 30, // 30 days
	}
	
	http.SetCookie(w, cookie)
}