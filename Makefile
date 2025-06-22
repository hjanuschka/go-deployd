.PHONY: build run test clean deps mongo-start mongo-stop mongo-status dashboard dashboard-dev dashboard-build e2e-test

# Build the application
build:
	go build -o bin/deployd cmd/deployd/main.go

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

# Run dashboard in development mode (separate from go-deployd)
dashboard-dev:
	@echo "🎨 Starting dashboard dev server..."
	@cd dashboard && npm install && npm run dev

# Run the application in development mode with MongoDB
run: mongo-start dashboard-build
	@echo "🚀 Starting go-deployd with dashboard..."
	@sleep 1
	go run cmd/deployd/main.go -dev

# Run with custom port
run-port:
	go run cmd/deployd/main.go -dev -port 3000

# Test the application
test:
	go test ./...

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

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

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