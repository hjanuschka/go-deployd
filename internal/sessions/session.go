package sessions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/hjanuschka/go-deployd/internal/database"
)

type SessionStore struct {
	store       database.StoreInterface
	development bool
}

type Session struct {
	ID          string                 `json:"id"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	development bool
}

func New(db database.DatabaseInterface, development bool) *SessionStore {
	return &SessionStore{
		store:       db.CreateStore("sessions"),
		development: development,
	}
}

func (ss *SessionStore) CreateSession(sessionID string) (*Session, error) {
	ctx := context.Background()
	
	if sessionID != "" {
		// Try to find existing session
		query := database.NewQueryBuilder().Where("id", "$eq", sessionID)
		existing, err := ss.store.FindOne(ctx, query)
		if err == nil && existing != nil {
			session := &Session{
				ID:          sessionID,
				Data:        make(map[string]interface{}),
				development: ss.development,
			}
			
			if data, exists := existing["data"]; exists {
				if dataMap, ok := data.(map[string]interface{}); ok {
					session.Data = dataMap
				} else {
					// Fallback for any unconverted data types
					session.Data = make(map[string]interface{})
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
	sessionDoc := map[string]interface{}{
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
	// Check explicit isRoot flag first (for system login, master key auth)
	if root, exists := s.Data["isRoot"]; exists {
		if rootBool, ok := root.(bool); ok {
			return rootBool
		}
	}
	
	// In development mode, only return true if no explicit isRoot flag is set
	// This allows proper authentication testing while maintaining dev convenience
	if s.development {
		return false // Changed: don't automatically grant root in dev mode
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
	
	query := database.NewQueryBuilder().Where("id", "$eq", s.ID)
	update := database.NewUpdateBuilder().
		Set("id", s.ID).
		Set("data", s.Data).
		Set("createdAt", s.CreatedAt).
		Set("updatedAt", s.UpdatedAt)
	
	_, err := store.store.Upsert(ctx, query, update)
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

