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
	
	// Extract field values using reflection with safe type assertions
	localCtx := &EventContext{
		Data:     safeGetMapField(v, "Data"),
		Query:    safeGetMapField(v, "Query"),
		Me:       safeGetMapField(v, "Me"),
		IsRoot:   safeGetBoolField(v, "IsRoot"),
		Internal: safeGetBoolField(v, "Internal"),
		Errors:   safeGetStringMapField(v, "Errors"),
		Cancel:   safeGetCancelField(v, "Cancel"),
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

// Safe helper functions for type conversion
func safeGetMapField(v reflect.Value, fieldName string) map[string]interface{} {
	val := getFieldValue(v, fieldName)
	if val == nil {
		return make(map[string]interface{})
	}
	if mapVal, ok := val.(map[string]interface{}); ok {
		return mapVal
	}
	return make(map[string]interface{})
}

func safeGetBoolField(v reflect.Value, fieldName string) bool {
	val := getFieldValue(v, fieldName)
	if val == nil {
		return false
	}
	if boolVal, ok := val.(bool); ok {
		return boolVal
	}
	return false
}

func safeGetStringMapField(v reflect.Value, fieldName string) map[string]string {
	val := getFieldValue(v, fieldName)
	if val == nil {
		return make(map[string]string)
	}
	if mapVal, ok := val.(map[string]string); ok {
		return mapVal
	}
	return make(map[string]string)
}

func safeGetCancelField(v reflect.Value, fieldName string) func(string, int) {
	val := getFieldValue(v, fieldName)
	if val == nil {
		return func(string, int) {} // no-op function
	}
	if cancelFunc, ok := val.(func(string, int)); ok {
		return cancelFunc
	}
	return func(string, int) {} // no-op function
}
`
	return fmt.Sprintf(template, userImports, userFunctions)
}