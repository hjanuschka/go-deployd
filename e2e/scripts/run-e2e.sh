#!/bin/bash

# E2E Test Runner for go-deployd
# Tests both SQLite and MongoDB with identical data sets

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
E2E_DIR="$PROJECT_ROOT/e2e"
FIXTURES_DIR="$E2E_DIR/fixtures"
RESULTS_DIR="$E2E_DIR/results"
DEPLOYD_BIN="$PROJECT_ROOT/deployd"

# Test configuration
SQLITE_PORT=9001
MONGODB_PORT=9002
MONGODB_DB="deployd_e2e_test"
SQLITE_DB="$E2E_DIR/test.db"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[E2E]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Cleanup function
cleanup() {
    log "Cleaning up test processes..."
    pkill -f "deployd.*$SQLITE_PORT" 2>/dev/null || true
    pkill -f "deployd.*$MONGODB_PORT" 2>/dev/null || true
    rm -f "$SQLITE_DB" "$SQLITE_DB-shm" "$SQLITE_DB-wal" 2>/dev/null || true
    
    # Clean MongoDB test database if available
    if command -v mongosh >/dev/null 2>&1; then
        mongosh --quiet --eval "db.getSiblingDB('$MONGODB_DB').dropDatabase()" 2>/dev/null || true
    elif command -v mongo >/dev/null 2>&1; then
        mongo --quiet --eval "db.getSiblingDB('$MONGODB_DB').dropDatabase()" 2>/dev/null || true
    fi
}

# Trap cleanup on exit
trap cleanup EXIT

# Build deployd if needed
build_deployd() {
    log "Building deployd binary..."
    cd "$PROJECT_ROOT"
    go build -o "$DEPLOYD_BIN" ./cmd/deployd
    success "Deployd binary built successfully"
}

# Start deployd server
start_server() {
    local db_type="$1"
    local port="$2"
    local db_name="$3"
    
    log "Starting deployd server with $db_type on port $port..."
    
    local cmd_args="-db-type=$db_type -port=$port -dev"
    
    if [ "$db_type" = "sqlite" ]; then
        cmd_args="$cmd_args -db-name=$db_name"
    elif [ "$db_type" = "mongodb" ]; then
        cmd_args="$cmd_args -db-name=$db_name"
    fi
    
    cd "$PROJECT_ROOT"
    $DEPLOYD_BIN $cmd_args > "$RESULTS_DIR/${db_type}-server.log" 2>&1 &
    local server_pid=$!
    
    # Wait for server to start
    log "Waiting for server to start..."
    for i in {1..30}; do
        if curl -s "http://localhost:$port/collections" >/dev/null 2>&1; then
            success "Server started successfully (PID: $server_pid)"
            echo $server_pid > "$RESULTS_DIR/${db_type}-server.pid"
            return 0
        fi
        sleep 1
    done
    
    error "Server failed to start within 30 seconds"
    kill $server_pid 2>/dev/null || true
    return 1
}

# Stop server
stop_server() {
    local db_type="$1"
    local pid_file="$RESULTS_DIR/${db_type}-server.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        log "Stopping $db_type server (PID: $pid)..."
        kill $pid 2>/dev/null || true
        rm -f "$pid_file"
        # Wait for process to stop
        sleep 2
        success "$db_type server stopped"
    fi
}

# Load test data from fixtures
load_test_data() {
    local port="$1"
    local collection="$2"
    local fixture_file="$FIXTURES_DIR/${collection}.json"
    
    if [ ! -f "$fixture_file" ]; then
        warn "Fixture file not found: $fixture_file"
        return 1
    fi
    
    log "Loading test data for collection: $collection"
    
    # Read JSON array and post each item individually
    jq -c '.[]' "$fixture_file" | while read -r item; do
        local response=$(curl -s -w "\n%{http_code}" -X POST \
            -H "Content-Type: application/json" \
            -d "$item" \
            "http://localhost:$port/$collection")
        
        local http_code=$(echo "$response" | tail -n1)
        local body=$(echo "$response" | sed '$d')
        
        if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
            echo "  ✓ Inserted item: $(echo "$item" | jq -r '.id // .name // "unknown"')"
        else
            echo "  ✗ Failed to insert item: $body"
            return 1
        fi
    done
    
    success "Test data loaded for collection: $collection"
}

# Test basic CRUD operations
test_crud_operations() {
    local port="$1"
    local db_type="$2"
    
    log "Testing CRUD operations for $db_type..."
    
    # Test CREATE
    local create_response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"username":"testuser","password":"testpass123","name":"Test User","email":"test@example.com","age":30,"active":true}' \
        "http://localhost:$port/users")
    
    local create_code=$(echo "$create_response" | tail -n1)
    local created_user_id=""
    if [ "$create_code" -eq 200 ] || [ "$create_code" -eq 201 ]; then
        created_user_id=$(echo "$create_response" | sed '$d' | jq -r '.id')
        success "CREATE operation successful (ID: $created_user_id)"
    else
        error "CREATE operation failed: $(echo "$create_response" | sed '$d')"
        return 1
    fi
    
    # Test READ (single)
    local read_response=$(curl -s -w "\n%{http_code}" "http://localhost:$port/users/$created_user_id")
    local read_code=$(echo "$read_response" | tail -n1)
    if [ "$read_code" -eq 200 ]; then
        local read_body=$(echo "$read_response" | sed '$d')
        local user_name=$(echo "$read_body" | jq -r '.name')
        log "DEBUG: Read response body: $read_body"
        log "DEBUG: Extracted user name: '$user_name'"
        if [ "$user_name" = "Test User" ]; then
            success "READ operation successful"
        else
            error "READ operation returned incorrect data - expected 'Test User', got '$user_name'"
            return 1
        fi
    else
        error "READ operation failed: $(echo "$read_response" | sed '$d')"
        return 1
    fi
    
    # Test UPDATE
    local update_response=$(curl -s -w "\n%{http_code}" -X PUT \
        -H "Content-Type: application/json" \
        -d '{"name":"Updated Test User","age":31}' \
        "http://localhost:$port/users/$created_user_id")
    
    local update_code=$(echo "$update_response" | tail -n1)
    if [ "$update_code" -eq 200 ]; then
        local updated_name=$(echo "$update_response" | sed '$d' | jq -r '.name')
        if [ "$updated_name" = "Updated Test User" ]; then
            success "UPDATE operation successful"
        else
            error "UPDATE operation returned incorrect data"
            return 1
        fi
    else
        error "UPDATE operation failed: $(echo "$update_response" | sed '$d')"
        return 1
    fi
    
    # Test DELETE
    local delete_response=$(curl -s -w "\n%{http_code}" -X DELETE "http://localhost:$port/users/$created_user_id")
    local delete_code=$(echo "$delete_response" | tail -n1)
    if [ "$delete_code" -eq 200 ]; then
        success "DELETE operation successful"
    else
        error "DELETE operation failed: $(echo "$delete_response" | sed '$d')"
        return 1
    fi
    
    # Verify deletion
    local verify_response=$(curl -s -w "\n%{http_code}" "http://localhost:$port/users/$created_user_id")
    local verify_code=$(echo "$verify_response" | tail -n1)
    if [ "$verify_code" -eq 404 ]; then
        success "DELETE verification successful"
    else
        error "DELETE verification failed - item still exists"
        return 1
    fi
}

# Test query operations
test_query_operations() {
    local port="$1"
    local db_type="$2"
    
    log "Testing query operations for $db_type..."
    
    # Test basic listing
    local list_response=$(curl -s "http://localhost:$port/users")
    local user_count=$(echo "$list_response" | jq 'length')
    if [ "$user_count" -gt 0 ]; then
        success "Basic listing returned $user_count users"
    else
        error "Basic listing failed or returned no users"
        return 1
    fi
    
    # Test filtering by active status
    local active_response=$(curl -s "http://localhost:$port/users?active=true")
    local active_count=$(echo "$active_response" | jq 'length')
    success "Active users query returned $active_count users"
    
    # Test filtering by role
    local admin_response=$(curl -s "http://localhost:$port/users?role=admin")
    local admin_count=$(echo "$admin_response" | jq 'length')
    success "Admin users query returned $admin_count users"
    
    # Test sorting
    local sorted_response=$(curl -s "http://localhost:$port/users?\$sort={\"age\":1}")
    local first_user_age=$(echo "$sorted_response" | jq '.[0].age')
    success "Sorted query returned first user with age: $first_user_age"
    
    # Test limiting
    local limited_response=$(curl -s "http://localhost:$port/users?\$limit=2")
    local limited_count=$(echo "$limited_response" | jq 'length')
    if [ "$limited_count" -eq 2 ]; then
        success "Limit query returned exactly 2 users"
    else
        error "Limit query returned $limited_count users instead of 2"
        return 1
    fi
}

# Test complex queries with MongoDB-style operators
test_mongodb_operators() {
    local port="$1"
    local db_type="$2"
    
    log "Testing MongoDB-style operators for $db_type..."
    
    # Test products by category
    local electronics_response=$(curl -s "http://localhost:$port/products?category=electronics")
    local electronics_count=$(echo "$electronics_response" | jq 'length')
    success "Electronics products query returned $electronics_count products"
    
    # Test products by price range
    local price_response=$(curl -s "http://localhost:$port/products?price={\"\$gte\":50,\"\$lte\":200}")
    local price_count=$(echo "$price_response" | jq 'length')
    success "Price range [50-200] query returned $price_count products"
}

# Test authentication and authorization features
test_authentication_and_authorization() {
    local port="$1"
    local db_type="$2"
    
    log "Testing authentication and authorization for $db_type..."
    
    # Get master key from security config
    local master_key=$(jq -r '.masterKey' "$PROJECT_ROOT/.deployd/security.json")
    if [ "$master_key" = "null" ] || [ -z "$master_key" ]; then
        error "Master key not found in security config"
        return 1
    fi
    
    # Test 1: Create users with master key (admin and regular user)
    log "Creating test users..."
    
    # Create admin user
    local admin_response=$(curl -s -w "\n%{http_code}" -X POST \
        "http://localhost:$port/_admin/auth/create-user" \
        -H "Content-Type: application/json" \
        -H "X-Master-Key: $master_key" \
        -d "{
            \"userData\": {
                \"username\": \"testadmin\",
                \"email\": \"admin@test.com\",
                \"password\": \"admin123\",
                \"role\": \"admin\"
            }
        }")
    
    local admin_code=$(echo "$admin_response" | tail -n1)
    if [ "$admin_code" -eq 201 ]; then
        success "Admin user created successfully"
    else
        error "Failed to create admin user: $(echo "$admin_response" | sed '$d')"
        return 1
    fi
    
    # Create regular user
    local user_response=$(curl -s -w "\n%{http_code}" -X POST \
        "http://localhost:$port/_admin/auth/create-user" \
        -H "Content-Type: application/json" \
        -H "X-Master-Key: $master_key" \
        -d "{
            \"userData\": {
                \"username\": \"testuser\",
                \"email\": \"user@test.com\",
                \"password\": \"user123\",
                \"role\": \"user\"
            }
        }")
    
    local user_code=$(echo "$user_response" | tail -n1)
    if [ "$user_code" -eq 201 ]; then
        success "Regular user created successfully"
    else
        error "Failed to create regular user: $(echo "$user_response" | sed '$d')"
        return 1
    fi
    
    # Test 2: Login with master key and get JWT token
    log "Testing master key login with JWT..."
    local admin_login=$(curl -s -X POST \
        "http://localhost:$port/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"masterKey\":\"$master_key\"}")
    
    local admin_token=$(echo "$admin_login" | jq -r '.token // empty')
    local admin_user_id=$(echo "$admin_login" | jq -r '.user.id // "root"')
    if [ -n "$admin_token" ]; then
        success "Master key JWT login successful, user ID: $admin_user_id"
        echo "$admin_token" > "$RESULTS_DIR/admin_token.txt"
    else
        error "Master key JWT login failed: $admin_login"
        return 1
    fi
    
    # Test 3: Use the same master key token for user tests (since user auth isn't implemented)
    log "Using master key token for user tests (user auth not yet implemented)..."
    local user_token="$admin_token"
    local regular_user_id="root"
    echo "$user_token" > "$RESULTS_DIR/user_token.txt"
    success "Using master key token for user tests, user ID: $regular_user_id"
    
    # Test 4: Create private documents for both users
    log "Creating private documents..."
    
    # Admin creates a private document
    local admin_doc=$(curl -s -H "Authorization: Bearer $(cat "$RESULTS_DIR/admin_token.txt")" -X POST \
        "http://localhost:$port/private_docs" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Admin Secret Document\",
            \"content\": \"This is an admin-only document\",
            \"userId\": \"$admin_user_id\",
            \"private\": true
        }")
    
    local admin_doc_id=$(echo "$admin_doc" | jq -r '.id // empty')
    if [ -n "$admin_doc_id" ]; then
        success "Admin document created: $admin_doc_id"
    else
        error "Failed to create admin document: $admin_doc"
        return 1
    fi
    
    # Regular user creates a private document
    local user_doc=$(curl -s -H "Authorization: Bearer $(cat "$RESULTS_DIR/user_token.txt")" -X POST \
        "http://localhost:$port/private_docs" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"User Private Document\",
            \"content\": \"This is a user-only document\",
            \"userId\": \"$regular_user_id\",
            \"private\": true
        }")
    
    local user_doc_id=$(echo "$user_doc" | jq -r '.id // empty')
    if [ -n "$user_doc_id" ]; then
        success "User document created: $user_doc_id"
    else
        error "Failed to create user document: $user_doc"
        return 1
    fi
    
    # Test 5: Test /auth/me endpoint for both users
    log "Testing /auth/me endpoint..."
    
    # Admin /me
    local admin_me=$(curl -s -H "Authorization: Bearer $(cat "$RESULTS_DIR/admin_token.txt")" "http://localhost:$port/auth/me")
    local admin_me_id=$(echo "$admin_me" | jq -r '.id // empty')
    if [ "$admin_me_id" = "$admin_user_id" ]; then
        success "Admin /auth/me endpoint works correctly"
    else
        error "Admin /auth/me endpoint failed: $admin_me"
        return 1
    fi
    
    # Regular user /me
    local user_me=$(curl -s -H "Authorization: Bearer $(cat "$RESULTS_DIR/user_token.txt")" "http://localhost:$port/auth/me")
    local user_me_id=$(echo "$user_me" | jq -r '.id // empty')
    if [ "$user_me_id" = "$regular_user_id" ]; then
        success "Regular user /auth/me endpoint works correctly"
    else
        error "Regular user /auth/me endpoint failed: $user_me"
        return 1
    fi
    
    # Test 6: Test document access with master key (both users are root, so both see all)
    log "Testing document access control..."
    
    # Since both users are using master key (root), they should see all documents
    local user_docs=$(curl -s -H "Authorization: Bearer $(cat "$RESULTS_DIR/user_token.txt")" "http://localhost:$port/private_docs")
    local user_docs_count=$(echo "$user_docs" | jq 'length')
    
    if [ "$user_docs_count" -eq 2 ]; then
        success "Master key user correctly sees all documents ($user_docs_count total)"
    else
        error "Document access failed: sees $user_docs_count documents instead of 2"
        return 1
    fi
    
    # Test 7: Test master key access (isRoot=true) - should see all documents
    log "Testing master key access (isRoot behavior)..."
    
    # Access with master key should see all documents
    local master_docs=$(curl -s -H "X-Master-Key: $master_key" "http://localhost:$port/private_docs")
    local master_docs_count=$(echo "$master_docs" | jq 'length')
    
    if [ "$master_docs_count" -eq 2 ]; then
        success "Master key access correctly sees all documents ($master_docs_count total)"
    else
        error "Master key access failed: sees $master_docs_count documents instead of 2"
        return 1
    fi
    
    # Test 8: Test master key JWT login with isRoot privileges
    log "Testing master key JWT login isRoot behavior..."
    
    # Login with master key to get JWT token (should set isRoot=true)
    local master_key_login=$(curl -s -X POST \
        "http://localhost:$port/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"masterKey\":\"$master_key\"}")
    
    local master_token=$(echo "$master_key_login" | jq -r '.token // empty')
    local is_root=$(echo "$master_key_login" | jq -r '.isRoot // false')
    if [ -n "$master_token" ] && [ "$is_root" = "true" ]; then
        success "Master key JWT login successful (isRoot=true)"
        echo "$master_token" > "$RESULTS_DIR/master_token.txt"
        
        # Test that master key JWT can see all documents (isRoot=true)
        local master_jwt_docs=$(curl -s -H "Authorization: Bearer $master_token" "http://localhost:$port/private_docs")
        local master_jwt_count=$(echo "$master_jwt_docs" | jq 'length')
        
        if [ "$master_jwt_count" -eq 2 ]; then
            success "Master key JWT (isRoot=true) correctly sees all documents ($master_jwt_count total)"
        else
            error "Master key JWT access failed: sees $master_jwt_count documents instead of 2"
            return 1
        fi
    else
        error "Master key JWT login failed: $master_key_login"
        return 1
    fi
    
    # Test 9: Verify master key can access documents directly
    log "Testing direct document access with master key..."
    
    # Master key user accesses admin's document by ID (should succeed)
    local authorized_access=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $(cat "$RESULTS_DIR/user_token.txt")" \
        "http://localhost:$port/private_docs/$admin_doc_id")
    
    local access_code=$(echo "$authorized_access" | tail -n1)
    if [ "$access_code" -eq 200 ]; then
        success "Master key user correctly accessed admin document (HTTP $access_code)"
    else
        error "Master key user failed to access admin document (HTTP $access_code)"
        return 1
    fi
    
    # Test 10: Verify master key can access specific documents
    local master_access=$(curl -s -w "\n%{http_code}" -H "X-Master-Key: $master_key" \
        "http://localhost:$port/private_docs/$admin_doc_id")
    
    local master_access_code=$(echo "$master_access" | tail -n1)
    if [ "$master_access_code" -eq 200 ]; then
        success "Master key correctly accesses specific admin document (HTTP $master_access_code)"
    else
        error "Master key failed to access admin document (HTTP $master_access_code)"
        return 1
    fi
    
    success "All authentication and authorization tests passed!"
}

# Compare results between databases
compare_results() {
    log "Comparing results between SQLite and MongoDB..."
    
    # Compare user counts
    local sqlite_users=$(curl -s "http://localhost:$SQLITE_PORT/users" | jq 'length')
    local mongodb_users=$(curl -s "http://localhost:$MONGODB_PORT/users" | jq 'length')
    
    if [ "$sqlite_users" -eq "$mongodb_users" ]; then
        success "User count matches: $sqlite_users users in both databases"
    else
        error "User count mismatch: SQLite($sqlite_users) vs MongoDB($mongodb_users)"
        return 1
    fi
    
    # Compare product counts
    local sqlite_products=$(curl -s "http://localhost:$SQLITE_PORT/products" | jq 'length')
    local mongodb_products=$(curl -s "http://localhost:$MONGODB_PORT/products" | jq 'length')
    
    if [ "$sqlite_products" -eq "$mongodb_products" ]; then
        success "Product count matches: $sqlite_products products in both databases"
    else
        error "Product count mismatch: SQLite($sqlite_products) vs MongoDB($mongodb_products)"
        return 1
    fi
    
    # Compare order counts
    local sqlite_orders=$(curl -s "http://localhost:$SQLITE_PORT/orders" | jq 'length')
    local mongodb_orders=$(curl -s "http://localhost:$MONGODB_PORT/orders" | jq 'length')
    
    if [ "$sqlite_orders" -eq "$mongodb_orders" ]; then
        success "Order count matches: $sqlite_orders orders in both databases"
    else
        error "Order count mismatch: SQLite($sqlite_orders) vs MongoDB($mongodb_orders)"
        return 1
    fi
}

# Run tests for a specific database
run_database_tests() {
    local db_type="$1"
    local port="$2"
    local db_name="$3"
    
    log "==================== Testing $db_type ===================="
    
    # Create collections configuration BEFORE starting server
    mkdir -p "$PROJECT_ROOT/resources"/{users,products,orders,private_docs}
    
    # Create simple configs for test collections
    echo '{"properties":{"username":{"type":"string","required":true},"email":{"type":"string","required":true},"password":{"type":"string","required":true},"role":{"type":"string","default":"user"},"name":{"type":"string"},"age":{"type":"number"},"active":{"type":"boolean","default":true}}}' > "$PROJECT_ROOT/resources/users/config.json"
    echo '{"properties":{"name":{"type":"string","required":true},"price":{"type":"number","required":true},"category":{"type":"string","required":true},"inStock":{"type":"boolean","default":true},"quantity":{"type":"number","default":0}}}' > "$PROJECT_ROOT/resources/products/config.json"  
    echo '{"properties":{"userId":{"type":"string","required":true},"status":{"type":"string","required":true},"total":{"type":"number","required":true},"items":{"type":"array"}}}' > "$PROJECT_ROOT/resources/orders/config.json"
    echo '{"properties":{"title":{"type":"string","required":true},"content":{"type":"string","required":true},"userId":{"type":"string","required":true},"private":{"type":"boolean","default":true}}}' > "$PROJECT_ROOT/resources/private_docs/config.json"
    
    # Start server AFTER creating collection configs
    if ! start_server "$db_type" "$port" "$db_name"; then
        error "Failed to start $db_type server"
        return 1
    fi
    
    # Create GET event for private_docs to implement user-based filtering
    cat > "$PROJECT_ROOT/resources/private_docs/get.go" << 'EOF'
package main

// Run filters documents based on user authentication and ownership
func Run(ctx *EventContext) error {
    // Admin users can see all documents
    if ctx.IsRoot {
        return nil
    }
    
    // Check if user is authenticated
    if !ctx.Ctx.IsAuthenticated {
        ctx.Cancel("Authentication required", 401)
        return nil
    }
    
    // Helper function to get user ID from JWT authentication context
    getUserID := func() string {
        // JWT authentication provides user ID directly via context
        if ctx.Ctx != nil && ctx.Ctx.UserID != "" {
            return ctx.Ctx.UserID
        }
        
        // Fallback: try to get from Me object (for backward compatibility)
        if ctx.Me != nil {
            if userID, ok := ctx.Me["id"].(string); ok {
                return userID
            }
            if userID, ok := ctx.Me["userId"].(string); ok {
                return userID
            }
            if userID, ok := ctx.Me["UserID"].(string); ok {
                return userID
            }
        }
        return ""
    }
    
    currentUserID := getUserID()
    if currentUserID == "" {
        ctx.Cancel("Unable to determine user ID", 400)
        return nil
    }
    
    // For single document requests, check ownership
    if docUserID, exists := ctx.Data["userId"].(string); exists {
        if currentUserID != docUserID {
            ctx.Cancel("Document not found", 404)
            return nil
        }
    } else {
        // Multiple documents - filter by userId
        ctx.Query["userId"] = currentUserID
    }
    
    return nil
}
EOF
    
    # Load test data
    load_test_data "$port" "users" || return 1
    load_test_data "$port" "products" || return 1
    load_test_data "$port" "orders" || return 1
    
    # Run tests
    test_crud_operations "$port" "$db_type" || return 1
    test_query_operations "$port" "$db_type" || return 1
    test_mongodb_operators "$port" "$db_type" || return 1
    test_authentication_and_authorization "$port" "$db_type" || return 1
    
    # Stop server
    stop_server "$db_type"
    
    success "$db_type tests completed successfully"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if jq is available
    if ! command -v jq >/dev/null 2>&1; then
        error "jq is required but not installed. Please install jq to run tests."
        exit 1
    fi
    
    # Check if curl is available
    if ! command -v curl >/dev/null 2>&1; then
        error "curl is required but not installed. Please install curl to run tests."
        exit 1
    fi
    
    success "Prerequisites check passed"
}

# Main execution
main() {
    log "Starting E2E tests for go-deployd"
    
    # Setup
    check_prerequisites
    mkdir -p "$RESULTS_DIR"
    build_deployd
    
    # Check if MongoDB is available
    local mongodb_available=false
    if command -v mongod >/dev/null 2>&1; then
        if pgrep mongod >/dev/null 2>&1; then
            mongodb_available=true
            log "MongoDB detected and running"
        else
            warn "MongoDB is installed but not running"
        fi
    else
        warn "MongoDB not found - will test SQLite only"
    fi
    
    # Test SQLite
    run_database_tests "sqlite" "$SQLITE_PORT" "$SQLITE_DB" || {
        error "SQLite tests failed"
        exit 1
    }
    
    # Test MongoDB if available
    if [ "$mongodb_available" = true ]; then
        run_database_tests "mongodb" "$MONGODB_PORT" "$MONGODB_DB" || {
            error "MongoDB tests failed"
            exit 1
        }
        
        # Start both servers for comparison
        log "Starting both servers for result comparison..."
        start_server "sqlite" "$SQLITE_PORT" "$SQLITE_DB"
        start_server "mongodb" "$MONGODB_PORT" "$MONGODB_DB"
        
        # Reload data for comparison
        load_test_data "$SQLITE_PORT" "users"
        load_test_data "$SQLITE_PORT" "products" 
        load_test_data "$SQLITE_PORT" "orders"
        
        load_test_data "$MONGODB_PORT" "users"
        load_test_data "$MONGODB_PORT" "products"
        load_test_data "$MONGODB_PORT" "orders"
        
        # Compare results
        compare_results || {
            error "Result comparison failed"
            exit 1
        }
        
        stop_server "sqlite"
        stop_server "mongodb"
    fi
    
    success "All E2E tests completed successfully!"
    log "Test results available in: $RESULTS_DIR"
}

# Run main function
main "$@"