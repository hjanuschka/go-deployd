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
    # Kill the dashboard dev server
    if [ ! -z "$DASHBOARD_PID" ]; then
        kill $DASHBOARD_PID 2>/dev/null
    fi
    # Kill any go run processes
    pkill -f "go run.*cmd/deployd/main.go" 2>/dev/null
    exit 0
}

# Set trap for cleanup
trap cleanup SIGINT SIGTERM

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

print_status "Starting development servers..."
print_status "🎨 Dashboard dev server will be available at: http://localhost:5173"
print_status "🚀 API server will be available at: http://localhost:2403"
print_status "📊 Dashboard (via server) will be available at: http://localhost:2403/_dashboard"
print_warning "Press Ctrl+C to stop all servers"
print_warning "Note: Manual restart required when Go files change (use 'make dev' for hot reload)"

# Start dashboard dev server in background
print_status "Starting React dashboard dev server..."
cd dashboard && npm run dev &
DASHBOARD_PID=$!
cd ..

# Wait a moment for dashboard to start
sleep 3

# Start Go server
print_status "Starting Go server..."
go run cmd/deployd/main.go -dev -db-type sqlite

# If we get here, the server exited, so cleanup
cleanup