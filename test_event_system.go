package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type TestResult struct {
	Name    string
	Success bool
	Message string
	Data    interface{}
}

func main() {
	baseURL := "http://localhost:2403"
	masterKey := os.Getenv("MASTER_KEY")
	if masterKey == "" {
		fmt.Println("❌ MASTER_KEY environment variable not set")
		os.Exit(1)
	}

	fmt.Println("🧪 Testing Event System for GET operations...")
	fmt.Println(strings.Repeat("=", 50))

	results := []TestResult{}

	// Test 1: Create a test user with sensitive data
	fmt.Println("\n📝 Test 1: Creating test user with sensitive data...")
	userData := map[string]interface{}{
		"username":          "testuser",
		"email":             "test@example.com",
		"password":          "secret123",
		"verificationToken": "super-secret-token",
		"role":              "user",
		"active":            true,
	}

	userID, success := createUser(baseURL, masterKey, userData)
	results = append(results, TestResult{
		Name:    "Create test user",
		Success: success,
		Message: fmt.Sprintf("User ID: %s", userID),
	})

	if !success {
		fmt.Println("❌ Failed to create user, stopping tests")
		return
	}

	// Test 2: Get single user (should hide password and verificationToken)
	fmt.Println("\n🔍 Test 2: Get single user by ID (should run GET event)...")
	singleUser, success := getUser(baseURL, masterKey, userID)
	results = append(results, TestResult{
		Name:    "Get single user with GET event",
		Success: success && singleUser["password"] == nil && singleUser["verificationToken"] == nil,
		Message: fmt.Sprintf("Password hidden: %v, Token hidden: %v",
			singleUser["password"] == nil, singleUser["verificationToken"] == nil),
		Data: singleUser,
	})

	// Test 3: Get all users (should hide password and verificationToken for each)
	fmt.Println("\n📋 Test 3: Get all users (should run GET event for each document)...")
	allUsers, success := getAllUsers(baseURL, masterKey)
	results = append(results, TestResult{
		Name:    "Get all users with GET events",
		Success: success && checkAllUsersHiddenFields(allUsers),
		Message: fmt.Sprintf("Found %d users, all have hidden fields", len(allUsers)),
		Data:    allUsers,
	})

	// Test 4: Get users with skipEvents (should NOT hide fields)
	fmt.Println("\n🚫 Test 4: Get users with $skipEvents=true (should NOT run GET events)...")
	rawUsers, success := getAllUsersRaw(baseURL, masterKey)
	results = append(results, TestResult{
		Name:    "Get users with skipEvents",
		Success: success && checkRawUsersHaveAllFields(rawUsers),
		Message: fmt.Sprintf("Found %d users with raw data", len(rawUsers)),
		Data:    rawUsers,
	})

	// Test 5: Verify event logging (check if GET event was called)
	fmt.Println("\n📊 Test 5: Verify GET event execution through data modification...")
	// The GET event should have run for tests 2 and 3, but not test 4
	eventRanCount := 0
	if singleUser != nil && checkEventModification(singleUser) {
		eventRanCount++
	}
	if allUsers != nil {
		for _, user := range allUsers {
			if userMap, ok := user.(map[string]interface{}); ok && checkEventModification(userMap) {
				eventRanCount++
				break // Count collection query as one event execution
			}
		}
	}

	results = append(results, TestResult{
		Name:    "Verify GET event execution",
		Success: eventRanCount >= 2, // Should run for single user + collection query
		Message: fmt.Sprintf("GET event executed %d times as expected", eventRanCount),
	})

	// Print results summary
	fmt.Println("\n" + "="*50)
	fmt.Println("📊 TEST RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 50))

	passedTests := 0
	for _, result := range results {
		status := "❌ FAIL"
		if result.Success {
			status = "✅ PASS"
			passedTests++
		}
		fmt.Printf("%s %s: %s\n", status, result.Name, result.Message)
	}

	fmt.Printf("\n🎯 Overall: %d/%d tests passed\n", passedTests, len(results))

	if passedTests == len(results) {
		fmt.Println("🎉 All tests passed! Event system is working correctly.")
		os.Exit(0)
	} else {
		fmt.Println("💥 Some tests failed. Check the event system implementation.")
		os.Exit(1)
	}
}

func createUser(baseURL, masterKey string, userData map[string]interface{}) (string, bool) {
	jsonData, _ := json.Marshal(userData)
	req, _ := http.NewRequest("POST", baseURL+"/users", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+masterKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error creating user: %v\n", err)
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Create user failed with status %d: %s\n", resp.StatusCode, string(body))
		return "", false
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if id, ok := result["id"].(string); ok {
		fmt.Printf("✅ Created user with ID: %s\n", id)
		return id, true
	}

	return "", false
}

func getUser(baseURL, masterKey, userID string) (map[string]interface{}, bool) {
	req, _ := http.NewRequest("GET", baseURL+"/users/"+userID, nil)
	req.Header.Set("Authorization", "Bearer "+masterKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error getting user: %v\n", err)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Get user failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil, false
	}

	var user map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&user)

	fmt.Printf("✅ Retrieved user: %s\n", user["email"])
	return user, true
}

func getAllUsers(baseURL, masterKey string) ([]interface{}, bool) {
	req, _ := http.NewRequest("GET", baseURL+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+masterKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error getting users: %v\n", err)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Get users failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil, false
	}

	var users []interface{}
	json.NewDecoder(resp.Body).Decode(&users)

	fmt.Printf("✅ Retrieved %d users\n", len(users))
	return users, true
}

func getAllUsersRaw(baseURL, masterKey string) ([]interface{}, bool) {
	req, _ := http.NewRequest("GET", baseURL+"/users?$skipEvents=true", nil)
	req.Header.Set("Authorization", "Bearer "+masterKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error getting raw users: %v\n", err)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Get raw users failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil, false
	}

	var users []interface{}
	json.NewDecoder(resp.Body).Decode(&users)

	fmt.Printf("✅ Retrieved %d raw users\n", len(users))
	return users, true
}

func checkAllUsersHiddenFields(users []interface{}) bool {
	if len(users) == 0 {
		return false
	}

	for _, user := range users {
		if userMap, ok := user.(map[string]interface{}); ok {
			if userMap["password"] != nil || userMap["verificationToken"] != nil {
				fmt.Printf("❌ User %s still has sensitive fields\n", userMap["email"])
				return false
			}
		}
	}

	fmt.Println("✅ All users have hidden sensitive fields")
	return true
}

func checkRawUsersHaveAllFields(users []interface{}) bool {
	if len(users) == 0 {
		return false
	}

	hasPasswordData := false
	for _, user := range users {
		if userMap, ok := user.(map[string]interface{}); ok {
			if userMap["password"] != nil {
				hasPasswordData = true
				break
			}
		}
	}

	if hasPasswordData {
		fmt.Println("✅ Raw users contain all fields including sensitive data")
		return true
	} else {
		fmt.Println("❌ Raw users are missing sensitive fields (events may have run)")
		return false
	}
}

func checkEventModification(user map[string]interface{}) bool {
	// Check if the GET event made any modifications that prove it ran
	// In our case, we're checking if sensitive fields were hidden
	return user["password"] == nil && user["verificationToken"] == nil
}
