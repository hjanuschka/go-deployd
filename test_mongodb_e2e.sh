#!/bin/bash

# MongoDB-only E2E test to verify the fix
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[E2E]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
MONGODB_PORT=9003
MONGODB_DB="mongodb_e2e_test"
PROJECT_ROOT="/Users/hjanuschka/go-deployd"
RESULTS_DIR="$PROJECT_ROOT/e2e/results"

# Start MongoDB server
log "Starting MongoDB server on port $MONGODB_PORT..."

# Clean up any existing server
pkill -f "deployd.*$MONGODB_PORT" 2>/dev/null || true
sleep 2

# Start deployd with MongoDB
./deployd -db-type=mongodb -db-name=$MONGODB_DB -port=$MONGODB_PORT -dev > mongodb_e2e.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
log "Waiting for server to start..."
for i in {1..30}; do
    if curl -s "http://localhost:$MONGODB_PORT/collections" >/dev/null 2>&1; then
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
MASTER_KEY=$(jq -r '.masterKey' ".deployd/security.json" 2>/dev/null)

# Create collections configuration
mkdir -p resources/{users,products,orders,private_docs}

echo '{"properties":{"username":{"type":"string","required":true},"email":{"type":"string","required":true},"password":{"type":"string","required":true},"role":{"type":"string","default":"user"},"name":{"type":"string"},"age":{"type":"number"},"active":{"type":"boolean","default":true}}}' > resources/users/config.json
echo '{"properties":{"name":{"type":"string","required":true},"price":{"type":"number","required":true},"category":{"type":"string","required":true},"inStock":{"type":"boolean","default":true},"quantity":{"type":"number","default":0}}}' > resources/products/config.json  
echo '{"properties":{"userId":{"type":"string","required":true},"status":{"type":"string","required":true},"total":{"type":"number","required":true},"items":{"type":"array"}}}' > resources/orders/config.json
echo '{"properties":{"title":{"type":"string","required":true},"content":{"type":"string","required":true},"userId":{"type":"string","required":true},"private":{"type":"boolean","default":true}}}' > resources/private_docs/config.json

# Test authentication and authorization
log "Testing authentication and authorization for MongoDB..."

# Create admin user
log "Creating admin user..."
ADMIN_RESP=$(curl -s -w "\n%{http_code}" -X POST \
    "http://localhost:$MONGODB_PORT/_admin/auth/create-user" \
    -H "Content-Type: application/json" \
    -H "X-Master-Key: $MASTER_KEY" \
    -d '{
        "userData": {
            "username": "testadmin",
            "email": "admin@test.com",
            "password": "admin123",
            "role": "admin"
        }
    }')

ADMIN_CODE=$(echo "$ADMIN_RESP" | tail -n1)
if [ "$ADMIN_CODE" -eq 201 ]; then
    success "Admin user created successfully"
else
    error "Failed to create admin user"
    exit 1
fi

# Create regular user
log "Creating regular user..."
USER_RESP=$(curl -s -w "\n%{http_code}" -X POST \
    "http://localhost:$MONGODB_PORT/_admin/auth/create-user" \
    -H "Content-Type: application/json" \
    -H "X-Master-Key: $MASTER_KEY" \
    -d '{
        "userData": {
            "username": "testuser",
            "email": "user@test.com",
            "password": "user123",
            "role": "user"
        }
    }')

USER_CODE=$(echo "$USER_RESP" | tail -n1)
if [ "$USER_CODE" -eq 201 ]; then
    success "Regular user created successfully"
else
    error "Failed to create regular user"
    exit 1
fi

# Login as admin
log "Testing admin login..."
ADMIN_LOGIN=$(curl -s -c admin_cookies.txt -X POST \
    "http://localhost:$MONGODB_PORT/users/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"testadmin","password":"admin123"}')

ADMIN_USER_ID=$(echo "$ADMIN_LOGIN" | jq -r '.id // empty')
if [ -n "$ADMIN_USER_ID" ]; then
    success "Admin login successful, user ID: $ADMIN_USER_ID"
else
    error "Admin login failed"
    exit 1
fi

# Login as regular user
log "Testing regular user login..."
USER_LOGIN=$(curl -s -c user_cookies.txt -X POST \
    "http://localhost:$MONGODB_PORT/users/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"user123"}')

REGULAR_USER_ID=$(echo "$USER_LOGIN" | jq -r '.id // empty')
if [ -n "$REGULAR_USER_ID" ]; then
    success "Regular user login successful, user ID: $REGULAR_USER_ID"
else
    error "Regular user login failed"
    exit 1
fi

# Test /me endpoint for both users
log "Testing /me endpoint..."

# Admin /me
ADMIN_ME=$(curl -s -b admin_cookies.txt "http://localhost:$MONGODB_PORT/users/me")
ADMIN_ME_ID=$(echo "$ADMIN_ME" | jq -r '.id // empty')
if [ "$ADMIN_ME_ID" = "$ADMIN_USER_ID" ]; then
    success "Admin /me endpoint works correctly"
else
    error "Admin /me endpoint failed"
    echo "Expected: $ADMIN_USER_ID, Got: $ADMIN_ME_ID"
    echo "Full response: $ADMIN_ME"
    exit 1
fi

# Regular user /me
USER_ME=$(curl -s -b user_cookies.txt "http://localhost:$MONGODB_PORT/users/me")
USER_ME_ID=$(echo "$USER_ME" | jq -r '.id // empty')
if [ "$USER_ME_ID" = "$REGULAR_USER_ID" ]; then
    success "Regular user /me endpoint works correctly"
else
    error "Regular user /me endpoint failed"
    echo "Expected: $REGULAR_USER_ID, Got: $USER_ME_ID"
    echo "Full response: $USER_ME"
    exit 1
fi

# Create private documents
log "Creating private documents..."

# Admin creates a document
ADMIN_DOC=$(curl -s -b admin_cookies.txt -X POST \
    "http://localhost:$MONGODB_PORT/private_docs" \
    -H "Content-Type: application/json" \
    -d "{
        \"title\": \"Admin Secret Document\",
        \"content\": \"This is an admin-only document\",
        \"userId\": \"$ADMIN_USER_ID\",
        \"private\": true
    }")

ADMIN_DOC_ID=$(echo "$ADMIN_DOC" | jq -r '.id // empty')
if [ -n "$ADMIN_DOC_ID" ]; then
    success "Admin document created: $ADMIN_DOC_ID"
else
    error "Failed to create admin document"
    exit 1
fi

# Regular user creates a document
USER_DOC=$(curl -s -b user_cookies.txt -X POST \
    "http://localhost:$MONGODB_PORT/private_docs" \
    -H "Content-Type: application/json" \
    -d "{
        \"title\": \"User Private Document\",
        \"content\": \"This is a user-only document\",
        \"userId\": \"$REGULAR_USER_ID\",
        \"private\": true
    }")

USER_DOC_ID=$(echo "$USER_DOC" | jq -r '.id // empty')
if [ -n "$USER_DOC_ID" ]; then
    success "User document created: $USER_DOC_ID"
else
    error "Failed to create user document"
    exit 1
fi

# Test document filtering
log "Testing document access control..."

# Regular user should only see their own documents
USER_DOCS=$(curl -s -b user_cookies.txt "http://localhost:$MONGODB_PORT/private_docs")
USER_DOCS_COUNT=$(echo "$USER_DOCS" | jq 'length')
USER_SEES_OWN=$(echo "$USER_DOCS" | jq --arg uid "$REGULAR_USER_ID" '.[] | select(.userId == $uid) | .id' | wc -l | tr -d ' ')

if [ "$USER_DOCS_COUNT" -eq 1 ] && [ "$USER_SEES_OWN" -eq 1 ]; then
    success "Regular user correctly sees only their own documents ($USER_DOCS_COUNT total)"
else
    error "Regular user document filtering failed: sees $USER_DOCS_COUNT documents, $USER_SEES_OWN are their own"
    echo "User documents: $USER_DOCS"
    exit 1
fi

# Master key should see all documents
MASTER_DOCS=$(curl -s -H "X-Master-Key: $MASTER_KEY" "http://localhost:$MONGODB_PORT/private_docs")
MASTER_DOCS_COUNT=$(echo "$MASTER_DOCS" | jq 'length')

if [ "$MASTER_DOCS_COUNT" -eq 2 ]; then
    success "Master key access correctly sees all documents ($MASTER_DOCS_COUNT total)"
else
    error "Master key access failed: sees $MASTER_DOCS_COUNT documents instead of 2"
    exit 1
fi

# Show logs
log "MongoDB server logs:"
cat mongodb_e2e.log

# Cleanup
kill $SERVER_PID
rm -f admin_cookies.txt user_cookies.txt mongodb_e2e.log

success "MongoDB E2E tests completed successfully!"