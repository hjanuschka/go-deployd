# External Library Support in Go-Deployd Event Scripts

Go-Deployd supports external libraries in both Go and JavaScript event scripts, but with different approaches.

## Go External Libraries

### ✅ **Full Support**: Any Go Module Available

Go event scripts have **full access** to the Go ecosystem via `go mod`. The compilation system automatically downloads and builds external dependencies.

#### Popular Libraries That Work:

```go
// UUID Generation
import "github.com/google/uuid"
uuid.New().String()

// Precise Decimal Math  
import "github.com/shopspring/decimal"
decimal.NewFromFloat(99.99).Mul(decimal.NewFromFloat(0.075))

// Cryptography
import "golang.org/x/crypto/bcrypt"
bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

// HTTP Requests
import "github.com/go-resty/resty/v2"
resty.New().R().Get("https://api.example.com")

// JSON Processing
import "github.com/tidwall/gjson"
gjson.Get(jsonString, "path.to.field")

// Time/Date Utilities
import "github.com/jinzhu/now"
now.BeginningOfDay()

// String Processing
import "github.com/iancoleman/strcase"
strcase.ToSnake("CamelCase") // -> "camel_case"

// Validation
import "github.com/go-playground/validator/v10"
validate.Struct(data)
```

#### Configuration:
```json
{
  "eventConfig": {
    "post": {
      "runtime": "go"
    }
  }
}
```

#### How It Works:
1. Server creates temporary `go.mod` with common dependencies
2. Your script imports any Go module
3. `go build -buildmode=plugin` compiles with full dependency resolution
4. Plugin loads at runtime with all external libraries available

---

## JavaScript External Libraries

### ⚠️ **Limited Support**: Built-in require() Modules Only

JavaScript event scripts use a **sandboxed runtime** with custom `require()` implementation. No npm install, but useful built-in modules are provided.

#### Available Modules:

```javascript
// Crypto utilities
const crypto = require('crypto');
crypto.randomUUID()           // Generate UUID-like strings
crypto.randomBytes(16)        // Generate random bytes

// Type checking utilities  
const util = require('util');
util.isArray(obj)            // Check if array
util.isObject(obj)           // Check if object

// Path utilities
const path = require('path');
path.extname('file.txt')     // Get extension: '.txt'
path.basename('/path/file')  // Get filename: 'file'
```

#### Built-in JavaScript Features:
```javascript
// All standard JavaScript works:
Array.map(), Object.keys(), String.methods()
Date, Math, JSON, RegExp
[...new Set(array)]          // ES6+ features
array.filter(x => x > 5)     // Arrow functions
```

#### Configuration:
```json
{
  "eventConfig": {
    "validate": {
      "runtime": "js"
    }
  }
}
```

---

## Comparison Table

| Feature | Go Scripts | JavaScript Scripts |
|---------|------------|-------------------|
| **External Libraries** | ✅ Full ecosystem (any Go module) | ⚠️ Built-in modules only |
| **HTTP Requests** | ✅ resty, http client libs | ❌ Not available |
| **Database Access** | ✅ SQL drivers, ORMs | ❌ Not available |
| **File System** | ✅ Full os/io access | ❌ Limited to built-ins |
| **Cryptography** | ✅ Full crypto libs | ⚠️ Basic crypto module |
| **Performance** | ✅ Compiled, very fast | ⚠️ Interpreted |
| **Setup Complexity** | ⚠️ Compilation required | ✅ Immediate execution |
| **Debugging** | ⚠️ Compilation errors | ✅ Runtime errors |

---

## Recommendations

### Use **Go** for:
- Complex business logic requiring external APIs
- Performance-critical operations  
- Advanced cryptography or data processing
- Integration with databases or external services
- Mathematical computations requiring precision

### Use **JavaScript** for:
- Simple data validation and transformation
- Quick prototyping and testing
- String/array manipulation
- Basic conditional logic
- When you need hot-reloading during development

---

## Examples

See the example files:
- `go_external_libs.go` - Comprehensive Go example with multiple libraries
- `js_require_modules.js` - JavaScript example using built-in modules

Both examples show real-world usage patterns and demonstrate the capabilities and limitations of each approach.