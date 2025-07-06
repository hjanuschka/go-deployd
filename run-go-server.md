# Running the Go Server Directly

## 1. Direct Run (simplest)
```bash
go run cmd/deployd/main.go -dev -db-type sqlite
```

## 2. Build and Run
```bash
# Build first
go build -o go-deployd cmd/deployd/main.go

# Then run
./go-deployd -dev -db-type sqlite
```

## 3. With Custom Port
```bash
go run cmd/deployd/main.go -dev -db-type sqlite -port 3000
```

## 4. With Environment Variables
```bash
PORT=3000 DB_TYPE=sqlite go run cmd/deployd/main.go -dev
```

## 5. Production Mode (without -dev flag)
```bash
go run cmd/deployd/main.go -db-type sqlite
```

## Available Flags:
- `-dev` - Development mode (enables CORS, debug logging)
- `-db-type` - Database type (sqlite, mongo)
- `-port` - Server port (default: 2403)
- `-mongo-url` - MongoDB connection URL
- `-data-dir` - Data directory path

## Quick Development Command:
```bash
# Kill any existing process on port 2403 and start fresh
lsof -ti:2403  < /dev/null |  xargs kill -9 2>/dev/null; go run cmd/deployd/main.go -dev -db-type sqlite
```
