#!/bin/bash

# Simple test script for the event system
set -e

BASE_URL="http://localhost:2403"
MASTER_KEY="${MASTER_KEY:-$(curl -s $BASE_URL/auth/master | jq -r '.token')}"

echo "🧪 Testing Event System"
echo "======================="
echo "Master Key: ${MASTER_KEY:0:20}..."
echo ""

# Test 1: Create a user
echo "📝 Test 1: Creating test user..."
USER_DATA='{
  "username": "eventtest",
  "email": "eventtest@example.com", 
  "password": "secret123",
  "verificationToken": "super-secret-token",
  "role": "user",
  "active": true
}'

USER_RESPONSE=$(curl -s -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $MASTER_KEY" \
  -d "$USER_DATA")

USER_ID=$(echo "$USER_RESPONSE" | jq -r '.id')
echo "✅ Created user with ID: $USER_ID"

# Test 2: Get single user (should hide sensitive fields)
echo ""
echo "🔍 Test 2: Getting single user (GET event should run)..."
SINGLE_USER=$(curl -s "$BASE_URL/users/$USER_ID" \
  -H "Authorization: Bearer $MASTER_KEY")

echo "Single user response:"
echo "$SINGLE_USER" | jq .

HAS_PASSWORD=$(echo "$SINGLE_USER" | jq 'has("password")')
HAS_TOKEN=$(echo "$SINGLE_USER" | jq 'has("verificationToken")')

if [ "$HAS_PASSWORD" = "false" ] && [ "$HAS_TOKEN" = "false" ]; then
  echo "✅ GET event worked: sensitive fields are hidden"
else
  echo "❌ GET event failed: sensitive fields still visible"
  echo "  Password present: $HAS_PASSWORD"
  echo "  Token present: $HAS_TOKEN"
fi

# Test 3: Get all users (should hide sensitive fields for each)
echo ""
echo "📋 Test 3: Getting all users (GET event should run for each)..."
ALL_USERS=$(curl -s "$BASE_URL/users" \
  -H "Authorization: Bearer $MASTER_KEY")

echo "All users response:"
echo "$ALL_USERS" | jq .

USER_COUNT=$(echo "$ALL_USERS" | jq 'length')
USERS_WITH_PASSWORD=$(echo "$ALL_USERS" | jq '[.[] | select(has("password"))] | length')

if [ "$USERS_WITH_PASSWORD" = "0" ]; then
  echo "✅ GET events worked for collection: no users have password field ($USER_COUNT users checked)"
else
  echo "❌ GET events failed for collection: $USERS_WITH_PASSWORD users still have password field"
fi

# Test 4: Get users with skipEvents (should NOT hide fields)
echo ""
echo "🚫 Test 4: Getting users with \$skipEvents=true (should NOT run GET events)..."
RAW_USERS=$(curl -s "$BASE_URL/users?%24skipEvents=true" \
  -H "Authorization: Bearer $MASTER_KEY")

echo "Raw users response:"
echo "$RAW_USERS" | jq .

RAW_USER_COUNT=$(echo "$RAW_USERS" | jq 'length')
RAW_USERS_WITH_PASSWORD=$(echo "$RAW_USERS" | jq '[.[] | select(has("password"))] | length')

if [ "$RAW_USERS_WITH_PASSWORD" -gt "0" ]; then
  echo "✅ Skip events worked: $RAW_USERS_WITH_PASSWORD users have password field (raw data preserved)"
else
  echo "❌ Skip events failed: no users have password field (events may have run anyway)"
fi

echo ""
echo "🎯 SUMMARY"
echo "=========="
echo "Test 1 (Create user): ✅"
echo "Test 2 (Single user GET event): $([ "$HAS_PASSWORD" = "false" ] && echo "✅" || echo "❌")"
echo "Test 3 (Collection GET events): $([ "$USERS_WITH_PASSWORD" = "0" ] && echo "✅" || echo "❌")"
echo "Test 4 (Skip events): $([ "$RAW_USERS_WITH_PASSWORD" -gt "0" ] && echo "✅" || echo "❌")"

# Cleanup
echo ""
echo "🧹 Cleaning up test user..."
curl -s -X DELETE "$BASE_URL/users/$USER_ID" \
  -H "Authorization: Bearer $MASTER_KEY" > /dev/null
echo "✅ Test user deleted"