package main

import (
	"strings"
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

// On POST - Modify data when creating documents (Go version)
func Run(ctx *events.EventContext) error {
    // Set defaults
    ctx.Data["createdAt"] = time.Now()
    if _, exists := ctx.Data["completed"]; !exists {
        ctx.Data["completed"] = false
    }
    
    // Add user info
    if ctx.Me != nil {
        ctx.Data["createdBy"] = ctx.Me["id"]
    }
    
    // Require authentication
    if ctx.Me == nil {
        ctx.Cancel("Authentication required", 401)
    }
    
    return nil
}

// Exported Run function for plugin
func Run(ctx *events.EventContext) error {
	return run(ctx)
}

// Rename user function to avoid conflicts
func run(ctx *events.EventContext) error {
	return nil
}