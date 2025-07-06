#!/bin/bash

# Create tmp directory if it doesn't exist
mkdir -p ./tmp

# Kill any existing go-deployd process on port 2403
echo "Checking for processes on port 2403..."
OLD_PIDS=$(lsof -ti:2403 2>/dev/null || true)
if [ ! -z "$OLD_PIDS" ]; then
    echo "Killing existing processes: $OLD_PIDS"
    echo "$OLD_PIDS" | xargs kill -9 2>/dev/null || true
    sleep 1
fi

# Build the server
echo "Building go-deployd..."
if ! go build -o ./tmp/main cmd/deployd/main.go; then
    echo "Build failed!"
    exit 1
fi

# Run the server
echo "Starting go-deployd..."
exec ./tmp/main -dev -db-type sqlite