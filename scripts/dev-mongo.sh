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

# Check if air is installed
if ! command -v air &> /dev/null; then
    print_error "Air is not installed. Installing air for Go hot reloading..."
    go install github.com/air-verse/air@latest
    if [ $? -ne 0 ]; then
        print_error "Failed to install air. Please install it manually:"
        print_error "go install github.com/air-verse/air@latest"
        exit 1
    fi
    print_status "Air installed successfully!"
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

# Start MongoDB if not running
if ! pgrep -f "mongod.*--dbpath .mongodb/data" > /dev/null; then
    print_status "Starting MongoDB..."
    make mongo-start
    if [ $? -ne 0 ]; then
        print_error "Failed to start MongoDB"
        exit 1
    fi
else
    print_status "MongoDB is already running"
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

# Create tmp directory for air
mkdir -p tmp

# Create air config for MongoDB
cat > .air-mongo.toml << EOF
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = ["-dev"]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main cmd/deployd/main.go"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "dashboard", ".mongodb", "bin", "e2e", "js-sandbox", "web"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = ["cmd", "internal", "resources"]
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
EOF

print_status "Starting development servers..."
print_status "🎨 Dashboard DEV (with hot reload): http://localhost:3001/_dashboard/"
print_status "🚀 API server will be available at: http://localhost:2403"
print_status "📊 Dashboard STATIC (no hot reload): http://localhost:2403/_dashboard"
print_status "🍃 MongoDB is running on port 27017"
print_warning "⚡ Use the Vite dev server URL for hot reloading!"
print_warning "Press Ctrl+C to stop all servers"

# Start dashboard dev server in background
print_status "Starting React dashboard dev server..."
(cd dashboard && npm run dev) &
DASHBOARD_PID=$!

# Wait a moment for dashboard to start
sleep 2

# Start Go server with hot reloading
print_status "Starting Go server with hot reloading..."
air -c .air-mongo.toml

# If we get here, air exited, so cleanup
cleanup