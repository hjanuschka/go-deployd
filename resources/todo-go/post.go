//go:build ignore
// +build ignore

package main

import (
	"time"
)

// Run processes todo after creation
func Run(ctx *EventContext) error {
	// Set default values if not provided
	if _, exists := ctx.Data["completed"]; !exists {
		ctx.Data["completed"] = false
	}

	if _, exists := ctx.Data["priority"]; !exists {
		ctx.Data["priority"] = 1
	}

	// Add creation timestamp if not set
	if _, exists := ctx.Data["createdAt"]; !exists {
		ctx.Data["createdAt"] = time.Now()
	}

	title, _ := ctx.Data["title"].(string)

	// Log todo creation using proper logging
	ctx.Log("Todo created successfully", map[string]interface{}{
		"title":     title,
		"action":    "post",
		"completed": ctx.Data["completed"],
		"priority":  ctx.Data["priority"],
	})

	return nil
}
