#!/bin/bash

# Test script for deployd-cli with JWT authentication

set -e

echo "Building CLI tool..."
go build -o deployd-cli ./cmd/deployd-cli

echo "Testing JWT-based CLI authentication..."

# Get master key from security config
MASTER_KEY=$(jq -r '.masterKey' .deployd/security.json)

echo "1. Testing login with master key..."
./deployd-cli -cmd=login -master-key="$MASTER_KEY"

echo -e "\n2. Testing GET all users..."
./deployd-cli -cmd=get -resource=users

echo -e "\n3. Testing POST new user..."
./deployd-cli -cmd=post -resource=users -data='{"name":"CLI Test User","email":"cli@test.com","active":true}'

echo -e "\n4. Testing GET with query..."
./deployd-cli -cmd=get -resource=users -id=$(./deployd-cli -cmd=get -resource=users | jq -r '.[0].id')

echo -e "\n5. Verifying JWT token was saved..."
if [ -f "$HOME/.deployd-token" ]; then
    echo "✓ Token file exists"
    echo "Token preview: $(head -c 50 $HOME/.deployd-token)..."
else
    echo "✗ Token file not found"
    exit 1
fi

echo -e "\nAll CLI tests passed! JWT authentication working correctly."