package main

import (
	"fmt"
	"reflect"
)

// Run validates file deletion
func Run(ctx *EventContext) error {
	// Only allow file owner or admin to delete (keep this enabled for security)
	if !ctx.IsRoot && ctx.Data["uploadedBy"] != ctx.Me["id"] {
		ctx.Cancel("You can only delete your own files", 403)
		return nil
	}

	// Example: Prevent deletion of files uploaded in the last hour (uncomment to enable)
	// if uploadedAt, ok := ctx.Data["uploadedAt"].(string); ok {
	// 	uploadTime, err := time.Parse(time.RFC3339, uploadedAt)
	// 	if err == nil {
	// 		hourAgo := time.Now().Add(-time.Hour)
	// 		if uploadTime.After(hourAgo) && !ctx.IsRoot {
	// 			ctx.Cancel("Cannot delete files uploaded within the last hour", 400)
	// 			return nil
	// 		}
	// 	}
	// }

	deletedBy := ""
	if ctx.Me != nil && ctx.Me["id"] != nil {
		deletedBy = fmt.Sprintf("%v", ctx.Me["id"])
	}

	ctx.Log("File deletion authorized", map[string]interface{}{
		"id":        ctx.Data["id"],
		"name":      ctx.Data["originalName"],
		"deletedBy": deletedBy,
	})

	return nil
}

// deployd provides utility functions for event handlers
var deployd = struct {
	// Log writes a message to the application logs
	Log func(message string, data ...map[string]interface{})
}{
	Log: func(message string, data ...map[string]interface{}) {
		// This is a fallback function - actual logging is handled by the runtime
		// when the context is properly set up in compile.go
	},
}

// EventContext provides context for event scripts
type EventContext struct {
	// Data is the document being processed
	Data map[string]interface{}
	
	// Query contains the query parameters
	Query map[string]interface{}
	
	// Me contains the current user (if authenticated)
	Me map[string]interface{}
	
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string
	
	// IsRoot indicates if the user has root/admin privileges
	IsRoot bool
	
	// Internal indicates if this is an internal request
	Internal bool
	
	// Errors holds validation errors
	Errors map[string]string
	
	// Cancel cancels the current operation with an error
	Cancel func(message string, statusCode int)
	
	// Log writes a message to the application logs
	Log func(message string, data ...map[string]interface{})
	
	// Emit sends real-time events to WebSocket clients
	// Usage: Emit("event-name", data) - broadcasts to all clients
	//        Emit("event-name", data, "room-name") - sends to specific room
	Emit func(event string, data interface{}, room ...string)
	
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
		Method:   safeGetStringField(v, "Method"),
		IsRoot:   safeGetBoolField(v, "IsRoot"),
		Internal: safeGetBoolField(v, "Internal"),
		Errors:   safeGetStringMapField(v, "Errors"),
		Cancel:   safeGetCancelField(v, "Cancel"),
		Log:      safeGetLogField(v, "Log"),
		Emit:     safeGetEmitField(v, "Emit"),
	}
	
	// Run the user's event handler
	err := Run(localCtx)
	
	// Sync changes back to the original context using reflection
	// Note: We need to work with the original pointer value, not the dereferenced struct
	origV := reflect.ValueOf(ctx)
	if origV.Kind() == reflect.Ptr && origV.Elem().Kind() == reflect.Struct {
		structV := origV.Elem()
		
		// Sync Data changes
		dataField := structV.FieldByName("Data")
		if dataField.IsValid() && dataField.CanSet() {
			dataField.Set(reflect.ValueOf(localCtx.Data))
		}
		
		// Sync hidden fields back
		hideFieldsField := structV.FieldByName("hideFields")
		if hideFieldsField.IsValid() && hideFieldsField.CanSet() {
			hideFieldsField.Set(reflect.ValueOf(localCtx.hideFields))
		}
	}
	
	return err
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

func safeGetStringField(v reflect.Value, fieldName string) string {
	val := getFieldValue(v, fieldName)
	if val == nil {
		return ""
	}
	if strVal, ok := val.(string); ok {
		return strVal
	}
	return ""
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

func safeGetLogField(v reflect.Value, fieldName string) func(string, ...map[string]interface{}) {
	val := getFieldValue(v, fieldName)
	if val == nil {
		return deployd.Log // fallback to global deployd.Log
	}
	if logFunc, ok := val.(func(string, ...map[string]interface{})); ok {
		return logFunc
	}
	return deployd.Log // fallback to global deployd.Log
}

func safeGetEmitField(v reflect.Value, fieldName string) func(string, interface{}, ...string) {
	val := getFieldValue(v, fieldName)
	if val == nil {
		return func(string, interface{}, ...string) {} // no-op function
	}
	if emitFunc, ok := val.(func(string, interface{}, ...string)); ok {
		return emitFunc
	}
	return func(string, interface{}, ...string) {} // no-op function
}
