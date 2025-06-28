#!/bin/bash

# Kill any process using port 2403
echo "Checking for processes on port 2403..."
lsof -ti:2403 | xargs -r kill -9 2>/dev/null || true

# Give it a moment to release the port
sleep 0.5

# Build and run the server
echo "Building and starting go-deployd..."
go build -o ./tmp/main cmd/deployd/main.go && ./tmp/main -dev -db-type sqlite