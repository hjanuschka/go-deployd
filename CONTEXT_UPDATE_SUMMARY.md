# Context Update Summary

## Changes Made

Updated all Go event handler test code in the `internal/events/*_test.go` files to remove the import of `"github.com/hjanuschka/go-deployd/internal/context"` from embedded Go code strings that represent event handlers.

### Files Modified:
1. `/Users/hjanuschka/go-deployd/internal/events/integration_test.go`
2. `/Users/hjanuschka/go-deployd/internal/events/manager_test.go`
3. `/Users/hjanuschka/go-deployd/internal/events/startup_plugins_test.go`

### Pattern Changed:

**From:**
```go
import (
    "github.com/hjanuschka/go-deployd/internal/context"
)

func Handle(ctx *context.Context, data map[string]interface{}) error {
```

**To:**
```go
// Context is a simple context for testing
type Context struct {
    Method string
}

func Handle(ctx *Context, data map[string]interface{}) error {
```

### Important Notes:
- Only the embedded Go code strings within test files were modified
- The test code itself still uses the actual `internal/context` package (as it should)
- This change makes the event handler code snippets in tests self-contained without needing internal package imports
- All tests are passing successfully after the changes

### Result:
The test event handlers now compile without needing the internal package imports, making them more portable and easier to test in isolation.