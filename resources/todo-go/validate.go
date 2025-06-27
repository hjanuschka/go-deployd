package main

import (
	"fmt"
	"strings"
)

// Run validates todo data before saving
func Run(ctx *EventContext) error {
	// Validate title
	title, ok := ctx.Data["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return ctx.Cancel("Title is required", 400)
	}
	
	if len(title) > 200 {
		return ctx.Cancel("Title is too long (max 200 characters)", 400)
	}
	
	// Validate priority if provided
	if priority, exists := ctx.Data["priority"]; exists {
		if priorityNum, ok := priority.(float64); ok {
			if priorityNum < 1 || priorityNum > 5 {
				return ctx.Cancel("Priority must be between 1 and 5", 400)
			}
		}
	}
	
	// Trim whitespace from title
	ctx.Data["title"] = strings.TrimSpace(title)
	
	fmt.Printf("Validated todo: %s\n", title)
	return nil
}