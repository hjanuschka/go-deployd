import (
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

// On POST - Modify data when creating documents (Go version with external library)
func Run(ctx *EventContext) error {
    // Generate unique ID using external UUID library
    uniqueID := uuid.New().String()
    ctx.Data["uniqueId"] = uniqueID
    ctx.Data["trackingId"] = "todo_" + uniqueID[:8]
    
    // Use decimal library for precise calculations
    if priorityVal, ok := ctx.Data["priority"].(float64); ok {
        priority := decimal.NewFromFloat(priorityVal)
        // Add some business logic with precise decimal math
        weight := priority.Mul(decimal.NewFromFloat(1.5))
        ctx.Data["weight"] = weight.InexactFloat64()
    }
    
    // Set creation timestamp
    ctx.Data["createdAt"] = "2025-01-01T00:00:00Z"
    ctx.Data["createdBy"] = "go-event-system"
    
    if _, exists := ctx.Data["completed"]; !exists {
        ctx.Data["completed"] = false
    }
    
    // Add user info if available
    if ctx.Me != nil {
        if userID, ok := ctx.Me["id"].(string); ok {
            ctx.Data["createdBy"] = userID
        }
    }
    
    // Require authentication for creating todos
    if ctx.Me == nil {
        ctx.Cancel("Authentication required to create todos", 401)
    }
    
    return nil
}