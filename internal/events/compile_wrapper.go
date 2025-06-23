package events

import (
	"fmt"
	"strings"
)

// CreateGoWrapper creates a wrapper that implements the plugin interface
func CreateGoWrapper(userCode string) string {
	// Parse user code to extract imports and functions separately
	lines := strings.Split(userCode, "\n")
	var imports []string
	var functions []string
	
	inImportBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip package declaration
		if strings.HasPrefix(trimmed, "package ") {
			continue
		}
		
		// Handle import statements
		if strings.HasPrefix(trimmed, "import ") {
			inImportBlock = true
			imports = append(imports, line)
		} else if inImportBlock && (trimmed == ")" || strings.Contains(trimmed, ")")) {
			imports = append(imports, line)
			inImportBlock = false
		} else if inImportBlock {
			imports = append(imports, line)
		} else if trimmed != "" {
			// Regular code (functions, etc.) - skip empty lines at the beginning
			functions = append(functions, line)
		}
	}
	
	userImports := strings.Join(imports, "\n")
	userFunctions := strings.Join(functions, "\n")
	template := `package main

import "reflect"

%s

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
	// Use reflection to extract field values since types don't match exactly
	v := reflect.ValueOf(ctx)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// Extract field values using reflection
	localCtx := &EventContext{
		Data:     getFieldValue(v, "Data").(map[string]interface{}),
		Query:    getFieldValue(v, "Query").(map[string]interface{}),
		Me:       getFieldValue(v, "Me").(map[string]interface{}),
		IsRoot:   getFieldValue(v, "IsRoot").(bool),
		Internal: getFieldValue(v, "Internal").(bool),
		Errors:   getFieldValue(v, "Errors").(map[string]string),
		Cancel:   getFieldValue(v, "Cancel").(func(string, int)),
	}
	
	return Run(localCtx)
}

// Helper function to get field value by name
func getFieldValue(v reflect.Value, fieldName string) interface{} {
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}
	return field.Interface()
}
`
	return fmt.Sprintf(template, userImports, userFunctions)
}