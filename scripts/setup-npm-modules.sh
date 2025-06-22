#!/bin/bash

# Setup npm modules for go-deployd JavaScript events
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
JS_SANDBOX="$PROJECT_ROOT/js-sandbox"

echo "🔧 Setting up npm modules for go-deployd..."

# Create js-sandbox directory if it doesn't exist
mkdir -p "$JS_SANDBOX"
cd "$JS_SANDBOX"

# Initialize package.json if it doesn't exist
if [ ! -f "package.json" ]; then
    echo "📦 Initializing package.json..."
    cat > package.json << 'EOF'
{
  "name": "go-deployd-js-sandbox",
  "version": "1.0.0",
  "description": "JavaScript sandbox for go-deployd event scripts",
  "main": "index.js",
  "scripts": {
    "install-common": "npm install lodash moment uuid validator axios crypto-js"
  },
  "keywords": ["deployd", "javascript", "sandbox"],
  "author": "go-deployd",
  "license": "MIT",
  "dependencies": {}
}
EOF
fi

# Install common npm packages that are useful for event scripts
echo "📦 Installing common npm packages..."

# Core utilities
npm install --save lodash@^4.17.21
npm install --save moment@^2.30.1
npm install --save uuid@^11.1.0
npm install --save validator@^13.15.15

# HTTP and crypto utilities
npm install --save axios@^1.6.0
npm install --save crypto-js@^4.2.0

# Data processing
npm install --save ramda@^0.29.0
npm install --save date-fns@^3.0.0

# String and number utilities
npm install --save numeral@^2.0.6
npm install --save slugify@^1.6.6

# JSON and data validation
npm install --save joi@^17.11.0
npm install --save ajv@^8.12.0

echo "✅ npm modules installed successfully!"
echo ""
echo "📋 Available modules for require() in JavaScript events:"
echo "  - lodash: Utility library"
echo "  - moment: Date manipulation"
echo "  - uuid: UUID generation"
echo "  - validator: String validation"
echo "  - axios: HTTP client"
echo "  - crypto-js: Cryptography"
echo "  - ramda: Functional programming"
echo "  - date-fns: Date utilities"
echo "  - numeral: Number formatting"
echo "  - slugify: URL-friendly strings"
echo "  - joi: Data validation"
echo "  - ajv: JSON schema validation"
echo ""
echo "🔧 Usage in JavaScript events:"
echo "  const _ = require('lodash');"
echo "  const moment = require('moment');"
echo "  const { v4: uuidv4 } = require('uuid');"
echo ""

# Create a test file showing how to use modules
cat > test-modules.js << 'EOF'
// Test file showing how to use npm modules in go-deployd events
const _ = require('lodash');
const moment = require('moment');
const { v4: uuidv4 } = require('uuid');
const validator = require('validator');

console.log('Testing npm modules:');
console.log('Lodash:', _.capitalize('hello world'));
console.log('Moment:', moment().format('YYYY-MM-DD'));
console.log('UUID:', uuidv4());
console.log('Validator:', validator.isEmail('test@example.com'));
EOF

echo "📄 Test file created: js-sandbox/test-modules.js"
echo "🎯 To add more packages: cd js-sandbox && npm install <package-name>"