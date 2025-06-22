package main

import (
    "time"
    "github.com/google/uuid"
)

// Go post event for testing data modification
func Run(ctx *EventContext) error {
    // Add creation metadata
    ctx.Data["createdBy"] = "go-event-system"
    ctx.Data["createdAt"] = time.Now().Format(time.RFC3339)
    ctx.Data["id"] = uuid.New().String()
    
    // Test user context
    if ctx.Me != nil {
        ctx.Data["createdByUser"] = ctx.Me["id"]
        ctx.Data["userRole"] = ctx.Me["role"]
    }
    
    // Test isRoot functionality
    if ctx.IsRoot {
        ctx.Data["createdByAdmin"] = true
        ctx.Data["adminPrivileges"] = []string{"read", "write", "delete"}
    }

    // Set default status if not provided
    if ctx.Data["status"] == nil {
        ctx.Data["status"] = "created"
    }

    return nil
}