package main

import (
    "fmt"
    "strings"
    "time"
    "math/rand"
)

// Process data when creating new pets (Go version)
func Run(ctx *EventContext) error {
    // Log the creation attempt
    deployd.Log("New pet creation started", map[string]interface{}{
        "name": ctx.Data["name"],
        "species": ctx.Data["species"],
        "user": ctx.Me,
    })
    
    // Auto-generate unique identifier if not provided
    if _, hasPetId := ctx.Data["petId"]; !hasPetId {
        petId := fmt.Sprintf("PET-%d-%s", time.Now().Unix(), generateRandomString(8))
        ctx.Data["petId"] = petId
    }
    
    // Set default values
    if _, hasVaccinated := ctx.Data["vaccinated"]; !hasVaccinated {
        ctx.Data["vaccinated"] = false
    }
    
    if _, hasSpayedNeutered := ctx.Data["spayedNeutered"]; !hasSpayedNeutered {
        ctx.Data["spayedNeutered"] = false
    }
    
    // Track creation metadata
    if ctx.Me != nil {
        ctx.Data["createdBy"] = ctx.Me["id"]
    } else {
        ctx.Data["createdBy"] = "anonymous"
    }
    ctx.Data["status"] = "active"
    
    // Validate user permissions - only authenticated users or root can create
    if ctx.Me == nil && !ctx.IsRoot {
        ctx.Cancel("Authentication required to create pets", 401)
        return nil
    }
    
    // Business logic - set adoption status
    if owner, hasOwner := ctx.Data["owner"].(string); hasOwner && strings.TrimSpace(owner) != "" {
        ctx.Data["adoptionStatus"] = "adopted"
        if _, hasAdoptionDate := ctx.Data["adoptionDate"]; !hasAdoptionDate {
            ctx.Data["adoptionDate"] = time.Now().Format(time.RFC3339)
        }
    } else {
        ctx.Data["adoptionStatus"] = "available"
    }
    
    // Generate microchip ID if not provided but pet is being adopted
    if adoptionStatus, _ := ctx.Data["adoptionStatus"].(string); adoptionStatus == "adopted" {
        if _, hasMicrochip := ctx.Data["microchipId"]; !hasMicrochip {
            microchipId := fmt.Sprintf("MC%d%s", time.Now().Unix(), generateRandomString(6))
            ctx.Data["microchipId"] = microchipId
        }
    }
    
    // Hide sensitive fields from being exposed in API responses
    ctx.Hide("createdBy")
    
    return nil
}

// Helper function to generate random string
func generateRandomString(length int) string {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[rand.Intn(len(charset))]
    }
    return string(b)
}