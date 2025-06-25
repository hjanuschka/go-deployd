package main

import (
	"encoding/json"
	"fmt"
)

// UserSessionData represents the session data stored for authenticated users
type UserSessionData struct {
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`
}

func main() {
	// Simulate what happens during session storage
	sessionData := UserSessionData{
		UserID:   "test-user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		Token:    "token123",
	}

	fmt.Printf("Original struct: %+v\n", sessionData)

	// Convert to JSON to see what fields are available
	jsonData, _ := json.Marshal(sessionData)
	fmt.Printf("JSON representation: %s\n", string(jsonData))

	// Convert back to map[string]interface{} to simulate what happens in EventContext
	var mapData map[string]interface{}
	json.Unmarshal(jsonData, &mapData)
	fmt.Printf("Map representation: %+v\n", mapData)

	// Check if userId and id fields are available
	if userId, ok := mapData["userId"].(string); ok {
		fmt.Printf("userId field found: %s\n", userId)
	} else {
		fmt.Printf("userId field NOT found\n")
	}

	if id, ok := mapData["id"].(string); ok {
		fmt.Printf("id field found: %s\n", id)
	} else {
		fmt.Printf("id field NOT found\n")
	}

	// Simulate the compatibility logic that should be in EventContext
	if userId, exists := mapData["userId"]; exists {
		mapData["id"] = userId
		fmt.Printf("Added id field for compatibility: %v\n", mapData["id"])
	}

	fmt.Printf("Final map: %+v\n", mapData)
}