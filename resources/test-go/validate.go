package main

import (
	"github.com/google/uuid"
	"strings"
	"time"
)

// Go validation event for testing data modification
func Run(ctx *EventContext) error {
	// Basic validation
	if title, ok := ctx.Data["title"].(string); !ok || strings.TrimSpace(title) == "" {
		ctx.Error("title", "Title is required")
		return nil
	}

	// Test data modification - add computed fields
	ctx.Data["validated"] = true
	ctx.Data["validatedAt"] = time.Now().Format(time.RFC3339)
	ctx.Data["validationId"] = uuid.New().String()

	// Test string manipulation
	if title, ok := ctx.Data["title"].(string); ok {
		ctx.Data["titleUpper"] = strings.ToUpper(title)
		ctx.Data["titleLength"] = len(title)
		ctx.Data["slug"] = strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	}

	// Test conditional logic
	if priority, ok := ctx.Data["priority"].(float64); ok {
		if priority >= 5 {
			ctx.Data["priorityLevel"] = "high"
		} else if priority >= 3 {
			ctx.Data["priorityLevel"] = "medium"
		} else {
			ctx.Data["priorityLevel"] = "low"
		}
	}

	// Test array/object modification
	if ctx.Data["tags"] == nil {
		ctx.Data["tags"] = []string{"go-validated"}
	}

	if ctx.Data["metadata"] == nil {
		ctx.Data["metadata"] = map[string]interface{}{
			"source":  "go-validation",
			"version": "1.0",
		}
	}

	return nil
}
