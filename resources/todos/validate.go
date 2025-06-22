package main

import (
	"github.com/hjanuschka/go-deployd/internal/events"
)

// Run validates the todo data
func Run(ctx *events.EventContext) error {
	// Simple validation - just check if title exists
	if title, ok := ctx.Data["title"].(string); !ok || title == "" {
		ctx.Error("title", "Title is required")
	}
	
	return nil
}