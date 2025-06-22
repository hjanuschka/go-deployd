package main

import (
    "time"
)

// Go put event for testing data modification
func Run(ctx *EventContext) error {
    // Add update metadata
    ctx.Data["updatedBy"] = "go-event-system"
    ctx.Data["updatedAt"] = time.Now().Format(time.RFC3339)
    ctx.Data["version"] = 2
    
    // Track what fields were modified (simplified)
    modifiedFields := []string{}
    
    if ctx.Data["title"] != nil {
        modifiedFields = append(modifiedFields, "title")
    }
    if ctx.Data["description"] != nil {
        modifiedFields = append(modifiedFields, "description")
    }
    if ctx.Data["priority"] != nil {
        modifiedFields = append(modifiedFields, "priority")
    }
    
    ctx.Data["modifiedFields"] = modifiedFields
    ctx.Data["modificationCount"] = len(modifiedFields)
    
    // Update status
    ctx.Data["status"] = "updated"

    return nil
}