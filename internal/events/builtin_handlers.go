package events

import (
	"fmt"
	"time"

	"github.com/hjanuschka/go-deployd/internal/context"
)

// BuiltinEventHandler represents a built-in event handler function
type BuiltinEventHandler func(*context.Context, map[string]interface{}) error

// BuiltinHandlers contains pre-compiled event handlers for testing
var BuiltinHandlers = map[string]BuiltinEventHandler{
	"test_modify": func(ctx *context.Context, data map[string]interface{}) error {
		// Modify data to prove event handlers work
		data["processed"] = true
		data["processedAt"] = time.Now().Format(time.RFC3339)
		data["handlerType"] = "builtin-go"
		
		if name, ok := data["name"].(string); ok {
			data["name"] = "Builtin Modified: " + name
		}
		
		// Add test metadata
		data["testMetadata"] = map[string]interface{}{
			"handler": "builtin_modify",
			"context": ctx.Method,
			"success": true,
		}
		
		return nil
	},
	
	"test_validate": func(ctx *context.Context, data map[string]interface{}) error {
		// Validate data and reject if invalid
		if name, ok := data["name"].(string); !ok || name == "" {
			return fmt.Errorf("builtin validation failed: name field is required and cannot be empty")
		}
		
		if value, ok := data["value"].(float64); !ok || value < 0 {
			return fmt.Errorf("builtin validation failed: value must be a non-negative number")
		}
		
		// Add validation metadata
		data["validated"] = true
		data["validatedBy"] = "builtin-handler"
		data["validatedAt"] = time.Now().Format(time.RFC3339)
		
		return nil
	},
	
	"test_enrichment": func(ctx *context.Context, data map[string]interface{}) error {
		// Add enrichment data
		data["enriched"] = true
		data["enrichedAt"] = time.Now().Format(time.RFC3339)
		data["userId"] = ctx.UserID
		data["method"] = ctx.Method
		data["isAuthenticated"] = ctx.IsAuthenticated
		
		// Add computed fields
		if value, ok := data["value"].(float64); ok {
			data["valueDoubled"] = value * 2
			data["valueSquared"] = value * value
		}
		
		return nil
	},
	
	"test_reject": func(ctx *context.Context, data map[string]interface{}) error {
		// Always reject to test error handling
		return fmt.Errorf("builtin handler intentionally rejected this request for testing")
	},
}

// RunBuiltinHandler executes a built-in event handler by name
func (usm *UniversalScriptManager) RunBuiltinHandler(handlerName string, ctx *context.Context, data map[string]interface{}) error {
	handler, exists := BuiltinHandlers[handlerName]
	if !exists {
		return fmt.Errorf("builtin handler '%s' not found", handlerName)
	}
	
	return handler(ctx, data)
}