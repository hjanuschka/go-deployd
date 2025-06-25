#!/bin/bash

# MySQL Runner Script for go-deployd
# Uses environment variables for configuration

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[MySQL Runner]${NC} $1"
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

# Load environment variables from .env file if it exists
load_env() {
    local env_file="$PROJECT_ROOT/.env"
    if [ -f "$env_file" ]; then
        log "Loading environment variables from .env file"
        # Export variables from .env file
        set -a  # automatically export all variables
        source "$env_file"
        set +a  # stop automatically exporting
        success "Environment variables loaded from .env"
    else
        warn ".env file not found at $env_file"
        warn "Using environment variables or defaults"
    fi
}

# Set default values for MySQL configuration
set_defaults() {
    # MySQL Configuration with fallbacks
    MYSQL_HOST="${MYSQL_HOST:-localhost}"
    MYSQL_PORT="${MYSQL_PORT:-3306}"
    MYSQL_USER="${MYSQL_USER:-root}"
    MYSQL_PASS="${MYSQL_PASS:-}"
    MYSQL_DB="${MYSQL_DB:-deployd}"
    
    # Server Configuration
    SERVER_PORT="${SERVER_PORT:-2403}"
    DEVELOPMENT_MODE="${DEVELOPMENT_MODE:-true}"
}

# Validate required environment variables
validate_config() {
    log "Validating MySQL configuration..."
    
    if [ -z "$MYSQL_HOST" ]; then
        error "MYSQL_HOST is required"
        return 1
    fi
    
    if [ -z "$MYSQL_USER" ]; then
        error "MYSQL_USER is required"
        return 1
    fi
    
    if [ -z "$MYSQL_DB" ]; then
        error "MYSQL_DB is required"
        return 1
    fi
    
    success "Configuration validated"
    log "MySQL Host: $MYSQL_HOST:$MYSQL_PORT"
    log "MySQL User: $MYSQL_USER"
    log "MySQL Database: $MYSQL_DB"
    log "Server Port: $SERVER_PORT"
}

# Build deployd binary
build_deployd() {
    log "Building deployd binary..."
    cd "$PROJECT_ROOT"
    
    if go build -o deployd ./cmd/deployd; then
        success "Deployd binary built successfully"
    else
        error "Failed to build deployd binary"
        return 1
    fi
}

# Start MySQL deployd server
start_mysql_server() {
    log "Starting deployd server with MySQL..."
    
    cd "$PROJECT_ROOT"
    
    # Build command arguments
    local cmd_args=(
        "-db-type=mysql"
        "-db-host=$MYSQL_HOST"
        "-db-port=$MYSQL_PORT"
        "-db-user=$MYSQL_USER"
        "-db-name=$MYSQL_DB"
        "-port=$SERVER_PORT"
    )
    
    # Add password if provided
    if [ -n "$MYSQL_PASS" ]; then
        cmd_args+=("-db-pass=$MYSQL_PASS")
    fi
    
    # Add development mode if enabled
    if [ "$DEVELOPMENT_MODE" = "true" ]; then
        cmd_args+=("-dev")
    fi
    
    log "Command: ./deployd ${cmd_args[*]}"
    log "Starting server..."
    
    # Start the server
    exec ./deployd "${cmd_args[@]}"
}

# Show usage information
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "MySQL Runner for go-deployd"
    echo ""
    echo "Configuration is loaded from environment variables or .env file:"
    echo "  MYSQL_HOST       MySQL host (default: localhost)"
    echo "  MYSQL_PORT       MySQL port (default: 3306)"
    echo "  MYSQL_USER       MySQL username (default: root)"
    echo "  MYSQL_PASS       MySQL password (default: empty)"
    echo "  MYSQL_DB         MySQL database name (default: deployd)"
    echo "  SERVER_PORT      Server port (default: 2403)"
    echo "  DEVELOPMENT_MODE Enable development mode (default: true)"
    echo ""
    echo "Options:"
    echo "  --help           Show this help message"
    echo "  --build-only     Only build the binary, don't start server"
    echo "  --check-config   Only validate configuration, don't start server"
    echo ""
    echo "Examples:"
    echo "  # Using .env file"
    echo "  $0"
    echo ""
    echo "  # Using environment variables"
    echo "  MYSQL_HOST=192.168.1.100 MYSQL_USER=myuser MYSQL_PASS=mypass $0"
    echo ""
    echo "  # Check configuration"
    echo "  $0 --check-config"
}

# Main execution
main() {
    local build_only=false
    local check_config=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help)
                show_usage
                exit 0
                ;;
            --build-only)
                build_only=true
                shift
                ;;
            --check-config)
                check_config=true
                shift
                ;;
            *)
                error "Unknown option: $1"
                echo ""
                show_usage
                exit 1
                ;;
        esac
    done
    
    log "Starting MySQL runner for go-deployd"
    
    # Load configuration
    load_env
    set_defaults
    validate_config || exit 1
    
    # Check config only mode
    if [ "$check_config" = true ]; then
        success "Configuration check completed"
        exit 0
    fi
    
    # Build binary
    build_deployd || exit 1
    
    # Build only mode
    if [ "$build_only" = true ]; then
        success "Build completed"
        exit 0
    fi
    
    # Start server
    start_mysql_server
}

# Run main function
main "$@"