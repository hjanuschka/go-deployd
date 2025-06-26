#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🧪 Running Go Unit Tests with Coverage..."
echo "========================================="

# Create coverage directory
mkdir -p coverage

# Run tests with coverage for all packages
echo -e "${YELLOW}Running tests with coverage...${NC}"
go test -v -coverprofile=coverage/coverage.out -covermode=atomic ./...

# Check if tests passed
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Tests failed!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All tests passed!${NC}"
echo ""

# Generate coverage report
echo -e "${YELLOW}Generating coverage report...${NC}"
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Show coverage summary by package
echo ""
echo -e "${YELLOW}Coverage Summary by Package:${NC}"
echo "----------------------------"
go test -coverprofile=coverage/coverage.out ./... | grep -E "coverage:|ok" | grep -v "no test files"

# Calculate coverage for critical packages
echo ""
echo -e "${YELLOW}Critical Package Coverage:${NC}"
echo "-------------------------"

# Function to check coverage for a package
check_coverage() {
    local package=$1
    local threshold=$2
    
    coverage=$(go test -cover ./$package 2>&1 | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//')
    
    if [ -z "$coverage" ]; then
        echo -e "  $package: ${RED}No tests found${NC}"
    else
        if (( $(echo "$coverage >= $threshold" | bc -l) )); then
            echo -e "  $package: ${GREEN}$coverage%${NC} (✓ meets $threshold% threshold)"
        else
            echo -e "  $package: ${RED}$coverage%${NC} (✗ below $threshold% threshold)"
        fi
    fi
}

# Check critical packages (50% threshold)
check_coverage "internal/auth" 50
check_coverage "internal/store" 50
check_coverage "internal/events" 50
check_coverage "internal/resources" 50
check_coverage "internal/router" 50

echo ""
echo -e "${GREEN}Coverage report generated at: coverage/coverage.html${NC}"
echo ""

# Run race condition detection
echo -e "${YELLOW}Running race condition detection...${NC}"
go test -race ./...

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ No race conditions detected!${NC}"
else
    echo -e "${RED}❌ Race conditions detected!${NC}"
fi

echo ""
echo "========================================="
echo -e "${GREEN}Test run complete!${NC}"