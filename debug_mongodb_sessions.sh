#!/bin/bash

# Quick test script to debug MongoDB session issue
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Start MongoDB server with deployd
log "Starting MongoDB server with deployd on port 9999..."

# Check if MongoDB is available
if ! command -v mongod >/dev/null 2>&1; then
    error "MongoDB not found - cannot test"
    exit 1
fi

if ! pgrep mongod >/dev/null 2>&1; then
    error "MongoDB is not running"
    exit 1
fi

# Clean up any existing server
pkill -f "deployd.*9999" 2>/dev/null || true
sleep 2

# Start deployd with MongoDB
./deployd -db-type=mongodb -db-name=debug_test -port=9999 -dev > mongodb_debug.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
log "Waiting for server to start..."
for i in {1..30}; do
    if curl -s "http://localhost:9999/collections" >/dev/null 2>&1; then
        success "Server started successfully"
        break
    fi
    sleep 1
done

if [ $i -eq 30 ]; then
    error "Server failed to start"
    exit 1
fi

# Get master key
MASTER_KEY=$(jq -r '.masterKey' ".deployd/security.json" 2>/dev/null || echo "mk_dev_test_key")
log "Using master key: $MASTER_KEY"

# Create a test user collection if it doesn't exist
mkdir -p resources/users
echo '{"properties":{"username":{"type":"string","required":true},"email":{"type":"string","required":true},"password":{"type":"string","required":true},"role":{"type":"string","default":"user"},"name":{"type":"string"}}}' > resources/users/config.json

# Test 1: Create a user with master key
log "Creating test user..."
CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    "http://localhost:9999/_admin/auth/create-user" \
    -H "Content-Type: application/json" \
    -H "X-Master-Key: $MASTER_KEY" \
    -d '{
        "userData": {
            "username": "debuguser2",
            "email": "debug2@test.com", 
            "password": "test123",
            "role": "admin"
        }
    }')

CREATE_CODE=$(echo "$CREATE_RESPONSE" | tail -n1)
if [ "$CREATE_CODE" -eq 201 ]; then
    success "User created successfully"
else
    error "Failed to create user: $CREATE_RESPONSE"
    cat mongodb_debug.log
    kill $SERVER_PID
    exit 1
fi

# Test 2: Login as the user
log "Logging in as test user..."
LOGIN_RESPONSE=$(curl -s -c debug_cookies.txt -X POST \
    "http://localhost:9999/users/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"debuguser2","password":"test123"}')

USER_ID=$(echo "$LOGIN_RESPONSE" | jq -r '.id // empty')
if [ -n "$USER_ID" ]; then
    success "Login successful, user ID: $USER_ID"
else
    error "Login failed: $LOGIN_RESPONSE"
    cat mongodb_debug.log
    kill $SERVER_PID
    exit 1
fi

# Test 3: Try /me endpoint
log "Testing /me endpoint..."
ME_RESPONSE=$(curl -s -w "\n%{http_code}" -b debug_cookies.txt "http://localhost:9999/users/me")
ME_CODE=$(echo "$ME_RESPONSE" | tail -n1)
ME_BODY=$(echo "$ME_RESPONSE" | sed '$d')

if [ "$ME_CODE" -eq 200 ]; then
    success "/me endpoint works: $ME_BODY"
else
    error "/me endpoint failed (HTTP $ME_CODE): $ME_BODY"
fi

log "Debug logs from server:"
cat mongodb_debug.log

# Cleanup
kill $SERVER_PID
rm -f debug_cookies.txt mongodb_debug.log

success "Debug test completed"