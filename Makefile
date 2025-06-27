.PHONY: help build run run_sqlite run_mysql test clean deps mongo-start mongo-stop mongo-status dashboard dashboard-dev dashboard-build e2e-test e2e-test-mysql dev dev-sqlite dev-mongo install-dev-tools

# Show help message with available targets
help:
	@echo "🚀 go-deployd Makefile Commands"
	@echo ""
	@echo "🏗️  Build Commands:"
	@echo "  make build                Build the binary"
	@echo "  make build-all           Build for multiple platforms"
	@echo "  make dist                Build standalone binary with dashboard"
	@echo "  make dashboard-build     Build dashboard for production (minified)"
	@echo "  make dashboard-build-dev Build dashboard for development (debug symbols)"
	@echo ""
	@echo "▶️  Run Commands:"
	@echo "  make run                 Run with MongoDB (requires MongoDB)"
	@echo "  make run_sqlite          Run with SQLite (no external DB required)"
	@echo "  make run_mysql           Run with MySQL (requires .env config)"
	@echo "  make dashboard-dev       Run dashboard in dev mode"
	@echo ""
	@echo "🔥 Development Commands:"
	@echo "  make dev                 Run both servers (React + Go) - recommended"
	@echo "  make dev-simple          Run both servers (no hot reload)"
	@echo "  make dev-sqlite          Run with hot reload (SQLite, requires air)"
	@echo "  make dev-mongo           Run with hot reload (MongoDB, requires air)"
	@echo "  make install-dev-tools   Install development tools (air, etc.)"
	@echo ""
	@echo "🧪 Test Commands:"
	@echo "  make test                Run Go tests"
	@echo "  make e2e-test            Run E2E tests (SQLite + MongoDB)"
	@echo "  make e2e-test-sqlite     Run E2E tests (SQLite only)"
	@echo "  make e2e-test-mysql      Run MySQL E2E tests (requires .env)"
	@echo ""
	@echo "🗄️  Database Commands:"
	@echo "  make mongo-start         Start local MongoDB"
	@echo "  make mongo-stop          Stop local MongoDB"
	@echo "  make mongo-status        Check MongoDB status"
	@echo ""
	@echo "🧹 Utility Commands:"
	@echo "  make clean               Clean build artifacts"
	@echo "  make clean-all           Clean everything (DB data, builds, etc.)"
	@echo "  make deps                Install/update Go dependencies"
	@echo "  make fmt                 Format Go code"
	@echo "  make lint                Lint Go code (requires golangci-lint)"
	@echo ""
	@echo "📚 Setup:"
	@echo "  For MySQL: cp .env.example .env && edit .env"
	@echo "  For development: make deps && make dashboard-build"

# Build the application
build:
	go build -o bin/deployd cmd/deployd/main.go

# Build standalone distribution with dashboard
dist: dashboard-build
	@echo "🚀 Building standalone binary..."
	go build -o bin/deployd-dist ./cmd/deployd
	@echo "✅ Standalone binary created: bin/deployd-dist"

# Start MongoDB with local data directory
mongo-start:
	@echo "🍃 Starting MongoDB..."
	@mkdir -p .mongodb/data .mongodb/logs
	@if pgrep -f "mongod.*--dbpath .mongodb/data" > /dev/null; then \
		echo "✅ MongoDB is already running"; \
	else \
		mongod --dbpath .mongodb/data --logpath .mongodb/logs/mongo.log --port 27017 --fork --quiet; \
		if [ $$? -eq 0 ]; then \
			echo "✅ MongoDB started on port 27017"; \
		else \
			echo "❌ Failed to start MongoDB. Trying without fork..."; \
			mongod --dbpath .mongodb/data --port 27017 --quiet & \
			sleep 2; \
			echo "✅ MongoDB started on port 27017 (background mode)"; \
		fi \
	fi

# Stop MongoDB
mongo-stop:
	@echo "🛑 Stopping MongoDB..."
	@pkill -f "mongod.*--dbpath .mongodb/data" || true
	@sleep 1
	@if pgrep -f "mongod.*--dbpath .mongodb/data" > /dev/null; then \
		echo "⚠️  MongoDB still running, force killing..."; \
		pkill -9 -f "mongod.*--dbpath .mongodb/data" || true; \
	fi
	@echo "✅ MongoDB stopped"

# Check MongoDB status
mongo-status:
	@echo "📊 MongoDB status:"
	@pgrep -f "mongod.*--dbpath .mongodb/data" > /dev/null && echo "✅ MongoDB is running" || echo "❌ MongoDB is not running"

# Build dashboard for production
dashboard-build:
	@echo "🏗️  Building dashboard..."
	@cd dashboard && npm install && npm run build
	@echo "✅ Dashboard built successfully"

# Build dashboard for development (with debug symbols)
dashboard-build-dev:
	@echo "🏗️  Building dashboard (development mode)..."
	@cd dashboard && npm install && npm run build:dev
	@echo "✅ Dashboard built successfully (with debug symbols)"

# Run dashboard in development mode (separate from go-deployd)
dashboard-dev:
	@echo "🎨 Starting dashboard dev server..."
	@cd dashboard && npm install && npm run dev

# Run the application in development mode with MongoDB
run: mongo-start dashboard-build
	@echo "🚀 Starting go-deployd with dashboard..."
	@sleep 1
	go run cmd/deployd/main.go -dev

# Run the application in development mode with SQLite (no MongoDB required)
run_sqlite: dashboard-build
	@echo "🚀 Starting go-deployd with SQLite and dashboard..."
	go run cmd/deployd/main.go -dev -db-type sqlite

# Run the application in development mode with MySQL (requires MySQL server and .env config)
run_mysql: dashboard-build
	@echo "🚀 Starting go-deployd with MySQL..."
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found. Please create .env file with MySQL configuration."; \
		echo "📝 Example:"; \
		echo "   cp .env.example .env"; \
		echo "   # Edit .env with your MySQL settings"; \
		exit 1; \
	fi
	@echo "📄 Loading configuration from .env file..."
	@./scripts/run_mysql.sh --check-config
	@echo "✅ Configuration validated. Starting server..."
	@./scripts/run_mysql.sh

# Run with custom port
run-port:
	go run cmd/deployd/main.go -dev -port 3000

# Test the application
test:
	go test ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@./run_tests.sh

# Run tests with coverage (verbose)
test-coverage-verbose:
	@echo "🧪 Running tests with coverage (verbose)..."
	@go test -v -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code (fallback to basic linting if golangci-lint fails)
lint:
	@echo "🔍 Running Go linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Using golangci-lint..."; \
		golangci-lint run || (echo "⚠️  golangci-lint failed, falling back to basic linting..."; \
		echo "Running go fmt..."; go fmt ./...; \
		echo "Running go vet (excluding resources)..."; go vet $$(go list ./... | grep -v '/resources/'); \
		echo "✅ Basic linting completed"); \
	else \
		echo "golangci-lint not found, using basic linting..."; \
		go fmt ./...; \
		go vet $$(go list ./... | grep -v '/resources/'); \
		echo "✅ Basic linting completed"; \
	fi

# Create binary for different platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/deployd-linux-amd64 cmd/deployd/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/deployd-darwin-amd64 cmd/deployd/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/deployd-windows-amd64.exe cmd/deployd/main.go

# Clean all (including MongoDB data and dashboard build)
clean-all: clean mongo-stop
	rm -rf .mongodb/ web/ dashboard/node_modules/ dashboard/dist/

# Quick demo with curl commands
demo: mongo-start
	@echo "🎬 Starting demo..."
	@sleep 1
	@go run cmd/deployd/main.go -dev &
	@sleep 3
	@echo "\n=== Creating a todo ==="
	@curl -X POST http://localhost:2403/todos \
		-H "Content-Type: application/json" \
		-d '{"title": "Learn Go", "completed": false, "priority": 1}' || true
	@echo "\n\n=== Getting all todos ==="
	@curl http://localhost:2403/todos || true
	@echo "\n\n=== Demo complete ==="
	@pkill -f "go run cmd/deployd/main.go" || true

# Run end-to-end tests across multiple databases
e2e-test:
	@echo "🧪 Running E2E tests..."
	@chmod +x e2e/scripts/run-e2e.sh
	@./e2e/scripts/run-e2e.sh

# Run E2E tests with MongoDB (requires MongoDB to be running)
e2e-test-with-mongo: mongo-start
	@echo "🧪 Running E2E tests with MongoDB..."
	@chmod +x e2e/scripts/run-e2e.sh
	@./e2e/scripts/run-e2e.sh

# Run E2E tests SQLite only (no MongoDB required)
e2e-test-sqlite:
	@echo "🧪 Running E2E tests (SQLite only)..."
	@chmod +x e2e/scripts/run-e2e.sh
	@./e2e/scripts/run-e2e.sh

# Run E2E tests for MySQL (requires MySQL server and .env config)
e2e-test-mysql:
	@echo "🧪 Running MySQL E2E tests..."
	@if [ ! -f .env ]; then \
		echo "❌ .env file not found. Please create .env file with MySQL configuration."; \
		echo "📝 Example:"; \
		echo "   cp .env.example .env"; \
		echo "   # Edit .env with your E2E_MYSQL_* settings"; \
		exit 1; \
	fi
	@chmod +x e2e/scripts/run-mysql-e2e.sh
	@./e2e/scripts/run-mysql-e2e.sh

# Install development tools
install-dev-tools:
	@echo "🔧 Installing development tools..."
	@echo "Installing air for Go hot reloading..."
	@go install github.com/air-verse/air@latest
	@if command -v air &> /dev/null; then \
		echo "✅ Air installed successfully"; \
	else \
		echo "❌ Failed to install air"; \
		exit 1; \
	fi
	@echo "✅ All development tools installed!"

# Development with both servers (recommended)
dev: dev-simple

# Development with both servers (no hot reload, but faster to start)
dev-simple:
	@echo "🔥 Starting development servers..."
	@echo "📝 Features:"
	@echo "   • React dashboard hot reload with Vite"
	@echo "   • Go server (manual restart needed for Go changes)"
	@echo "   • SQLite database (no external dependencies)"
	@echo ""
	@chmod +x scripts/dev-simple.sh
	@./scripts/dev-simple.sh

# Development with hot reload using SQLite
dev-sqlite: dashboard-build-dev
	@echo "🔥 Starting development servers with hot reload (SQLite)..."
	@echo "📝 Features:"
	@echo "   • Go server hot reload with Air"
	@echo "   • React dashboard hot reload with Vite"
	@echo "   • SQLite database (no external dependencies)"
	@echo "   • Dashboard built with debug symbols and sourcemaps"
	@echo ""
	@chmod +x scripts/dev.sh
	@./scripts/dev.sh

# Development with hot reload using MongoDB
dev-mongo: dashboard-build-dev
	@echo "🔥 Starting development servers with hot reload (MongoDB)..."
	@echo "📝 Features:"
	@echo "   • Go server hot reload with Air"
	@echo "   • React dashboard hot reload with Vite"
	@echo "   • MongoDB database"
	@echo "   • Dashboard built with debug symbols and sourcemaps"
	@echo ""
	@chmod +x scripts/dev-mongo.sh
	@./scripts/dev-mongo.sh