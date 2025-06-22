package events

import (
	"fmt"
	"strings"
)

// createGoWrapper creates a wrapper that implements the plugin interface
func createGoWrapper(userCode string) string {
	// Remove package declaration from user code
	lines := strings.Split(userCode, "\n")
	var filteredLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	userCode = strings.Join(filteredLines, "\n")
	template := `package main

// EventContext provides context for event scripts
type EventContext struct {
	// Data is the document being processed
	Data map[string]interface{}
	
	// Query contains the query parameters
	Query map[string]interface{}
	
	// Me contains the current user (if authenticated)
	Me map[string]interface{}
	
	// IsRoot indicates if the user has root/admin privileges
	IsRoot bool
	
	// Internal indicates if this is an internal request
	Internal bool
	
	// Errors holds validation errors
	Errors map[string]string
	
	// Cancel cancels the current operation with an error
	Cancel func(message string, statusCode int)
	
	// Hide removes a field from the response
	hideFields []string
}

// Error adds a validation error
func (ctx *EventContext) Error(field, message string) {
	if ctx.Errors == nil {
		ctx.Errors = make(map[string]string)
	}
	ctx.Errors[field] = message
}

// HasErrors returns true if there are validation errors
func (ctx *EventContext) HasErrors() bool {
	return len(ctx.Errors) > 0
}

// Hide removes a field from the response
func (ctx *EventContext) Hide(field string) {
	ctx.hideFields = append(ctx.hideFields, field)
	// Also remove from data
	delete(ctx.Data, field)
}

// GetHiddenFields returns the list of fields to hide
func (ctx *EventContext) GetHiddenFields() []string {
	return ctx.hideFields
}

// User code starts here
%s
// User code ends here

// EventHandler is the exported plugin handler
var EventHandler eventHandler

type eventHandler struct{}

// Run implements the plugin interface
func (h eventHandler) Run(ctx interface{}) error {
	// Convert the interface to our EventContext
	eventCtx := ctx.(*EventContext)
	return Run(eventCtx)
}
`
	return fmt.Sprintf(template, userCode)
}