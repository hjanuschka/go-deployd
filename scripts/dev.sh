#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[DEV]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to cleanup processes on exit
cleanup() {
    print_status "Cleaning up processes..."
    jobs -p | xargs -r kill
    exit 0
}

# Set trap for cleanup
trap cleanup SIGINT SIGTERM

# Check if nodemon is available
if ! command -v npx nodemon &> /dev/null; then
    print_warning "Nodemon is not installed. Installing development dependencies..."
    npm install
    if [ $? -ne 0 ]; then
        print_warning "Failed to install nodemon automatically. You can:"
        print_warning "1. Install manually: npm install"
        print_warning "2. Or use regular mode: make run_sqlite"
        print_warning ""
        print_status "Falling back to regular go run mode..."
        USE_NODEMON=false
    else
        print_status "Nodemon installed successfully!"
        USE_NODEMON=true
    fi
else
    USE_NODEMON=true
fi

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "This script must be run from the project root directory"
    exit 1
fi

# Check if dashboard directory exists
if [ ! -d "dashboard" ]; then
    print_error "Dashboard directory not found"
    exit 1
fi

# Install dashboard dependencies if needed
if [ ! -d "dashboard/node_modules" ]; then
    print_status "Installing dashboard dependencies..."
    cd dashboard && npm install
    if [ $? -ne 0 ]; then
        print_error "Failed to install dashboard dependencies"
        exit 1
    fi
    cd ..
fi

# Create tmp directory for builds
mkdir -p tmp

# Initial dashboard build
print_status "Building dashboard initially..."
cd dashboard && npm run build && cd ..
if [ $? -ne 0 ]; then
    print_error "Initial dashboard build failed. Exiting."
    exit 1
fi

print_status "Starting development with auto-rebuild..."
print_status "🚀 Server: http://localhost:2403"
print_status "📊 Dashboard: http://localhost:2403/_dashboard"
print_status "⚡ Both Go and React files will auto-rebuild with Nodemon"
print_warning "Press Ctrl+C to stop the server"

# Start Go server with hot reloading
if [ "$USE_NODEMON" = true ]; then
    print_status "Starting Go server with hot reloading..."
    npm run dev
else
    print_status "Starting Go server (without hot reloading)..."
    go run cmd/deployd/main.go -dev -db-type sqlite
fi

# If we get here, the server exited, so cleanup
cleanup