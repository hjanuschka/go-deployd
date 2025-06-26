package main

import (
	"strings"
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

func run(ctx *events.EventContext) error { return nil }

// Exported Run function for plugin
func Run(ctx *events.EventContext) error {
	return run(ctx)
}

// Rename user function to avoid conflicts
func run(ctx *events.EventContext) error {
	return nil
}