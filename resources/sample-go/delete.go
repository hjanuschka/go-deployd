package main

import (
    "strings"
)

// Process data before deleting pets (Go version)
func Run(ctx *EventContext) error {
    // Validate permissions - only authenticated users or root can delete
    if ctx.Me == nil && !ctx.IsRoot {
        ctx.Cancel("Authentication required to delete pets", 401)
        return nil
    }
    
    // Prevent deletion of adopted pets without special permission
    if adoptionStatus, hasStatus := ctx.Data["adoptionStatus"].(string); hasStatus && adoptionStatus == "adopted" && !ctx.IsRoot {
        ctx.Cancel("Cannot delete adopted pets. Please contact an administrator.", 403)
        return nil
    }
    
    // Additional protection for pets with microchips
    if microchipId, hasMicrochip := ctx.Data["microchipId"].(string); hasMicrochip && microchipId != "" && !ctx.IsRoot {
        ctx.Cancel("Cannot delete microchipped pets. Please contact an administrator.", 403)
        return nil
    }
    
    // Business logic - check if pet has important records
    if vetNotes, hasNotes := ctx.Data["vetNotes"].(string); hasNotes && strings.Contains(strings.ToLower(vetNotes), "medical") {
        // Log warning - but we can't use Console
    }
    
    // Soft delete implementation - uncomment to enable
    // Instead of actual deletion, mark as inactive and preserve data
    /*
    ctx.Data["status"] = "deleted"
    ctx.Data["deletedAt"] = time.Now().Format(time.RFC3339)
    if ctx.Me != nil {
        ctx.Data["deletedBy"] = ctx.Me["id"]
    } else {
        ctx.Data["deletedBy"] = "root"
    }
    return ctx.Cancel("Pet marked as deleted instead of permanent deletion")
    */
    
    // Pet deletion authorized
    
    return nil
}