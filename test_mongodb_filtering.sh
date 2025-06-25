#!/bin/bash

# Quick test to verify MongoDB document filtering works
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Start MongoDB server with deployd
log "Starting MongoDB server with deployd on port 9998..."

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
pkill -f "deployd.*9998" 2>/dev/null || true
sleep 2

# Start deployd with MongoDB
./deployd -db-type=mongodb -db-name=filter_test -port=9998 -dev > mongodb_filter.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
log "Waiting for server to start..."
for i in {1..30}; do
    if curl -s "http://localhost:9998/collections" >/dev/null 2>&1; then
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

# Create test collections
mkdir -p resources/{users,private_docs}
echo '{"properties":{"username":{"type":"string","required":true},"email":{"type":"string","required":true},"password":{"type":"string","required":true},"role":{"type":"string","default":"user"}}}' > resources/users/config.json
echo '{"properties":{"title":{"type":"string","required":true},"content":{"type":"string","required":true},"userId":{"type":"string","required":true},"private":{"type":"boolean","default":true}}}' > resources/private_docs/config.json

# Create users
log "Creating test users..."
USER1_RESP=$(curl -s -X POST "http://localhost:9998/_admin/auth/create-user" \
    -H "Content-Type: application/json" \
    -H "X-Master-Key: $MASTER_KEY" \
    -d '{"userData": {"username": "user1", "email": "user1@test.com", "password": "test123", "role": "user"}}')

USER2_RESP=$(curl -s -X POST "http://localhost:9998/_admin/auth/create-user" \
    -H "Content-Type: application/json" \
    -H "X-Master-Key: $MASTER_KEY" \
    -d '{"userData": {"username": "user2", "email": "user2@test.com", "password": "test123", "role": "user"}}')

log "Users created"

# Login as user1
log "Logging in as user1..."
USER1_LOGIN=$(curl -s -c user1_cookies.txt -X POST "http://localhost:9998/users/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"user1","password":"test123"}')

USER1_ID=$(echo "$USER1_LOGIN" | jq -r '.id')
success "User1 logged in with ID: $USER1_ID"

# Login as user2  
log "Logging in as user2..."
USER2_LOGIN=$(curl -s -c user2_cookies.txt -X POST "http://localhost:9998/users/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"user2","password":"test123"}')

USER2_ID=$(echo "$USER2_LOGIN" | jq -r '.id')
success "User2 logged in with ID: $USER2_ID"

# User1 creates a document
log "User1 creating a private document..."
USER1_DOC=$(curl -s -b user1_cookies.txt -X POST "http://localhost:9998/private_docs" \
    -H "Content-Type: application/json" \
    -d "{\"title\":\"User1 Document\",\"content\":\"This belongs to user1\",\"userId\":\"$USER1_ID\",\"private\":true}")

USER1_DOC_ID=$(echo "$USER1_DOC" | jq -r '.id')
success "User1 document created: $USER1_DOC_ID"

# User2 creates a document
log "User2 creating a private document..."
USER2_DOC=$(curl -s -b user2_cookies.txt -X POST "http://localhost:9998/private_docs" \
    -H "Content-Type: application/json" \
    -d "{\"title\":\"User2 Document\",\"content\":\"This belongs to user2\",\"userId\":\"$USER2_ID\",\"private\":true}")

USER2_DOC_ID=$(echo "$USER2_DOC" | jq -r '.id')
success "User2 document created: $USER2_DOC_ID"

# Test: User1 should only see their own document
log "Testing user1 document filtering..."
USER1_DOCS=$(curl -s -b user1_cookies.txt "http://localhost:9998/private_docs")
USER1_DOC_COUNT=$(echo "$USER1_DOCS" | jq 'length')

if [ "$USER1_DOC_COUNT" -eq 1 ]; then
    success "User1 correctly sees only their own document ($USER1_DOC_COUNT total)"
else
    error "User1 document filtering failed: sees $USER1_DOC_COUNT documents instead of 1"
    echo "User1 sees: $USER1_DOCS"
fi

# Test: User2 should only see their own document
log "Testing user2 document filtering..."
USER2_DOCS=$(curl -s -b user2_cookies.txt "http://localhost:9998/private_docs")
USER2_DOC_COUNT=$(echo "$USER2_DOCS" | jq 'length')

if [ "$USER2_DOC_COUNT" -eq 1 ]; then
    success "User2 correctly sees only their own document ($USER2_DOC_COUNT total)"
else
    error "User2 document filtering failed: sees $USER2_DOC_COUNT documents instead of 1"
    echo "User2 sees: $USER2_DOCS"
fi

# Test: Master key should see all documents
log "Testing master key access..."
MASTER_DOCS=$(curl -s -H "X-Master-Key: $MASTER_KEY" "http://localhost:9998/private_docs")
MASTER_DOC_COUNT=$(echo "$MASTER_DOCS" | jq 'length')

if [ "$MASTER_DOC_COUNT" -eq 2 ]; then
    success "Master key correctly sees all documents ($MASTER_DOC_COUNT total)"
else
    error "Master key access failed: sees $MASTER_DOC_COUNT documents instead of 2"
    echo "Master sees: $MASTER_DOCS"
fi

# Show debug logs
log "Debug logs:"
cat mongodb_filter.log

# Cleanup
kill $SERVER_PID
rm -f user1_cookies.txt user2_cookies.txt mongodb_filter.log

success "MongoDB document filtering test completed"