package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Process data when updating pets (Go version)
func Run(ctx *EventContext) error {
	// Track update metadata
	if ctx.Me != nil {
		ctx.Data["updatedBy"] = ctx.Me["id"]
	} else {
		ctx.Data["updatedBy"] = "anonymous"
	}
	ctx.Data["updatedAt"] = time.Now().Format(time.RFC3339)

	// Validate user permissions - only authenticated users can update
	if ctx.Me == nil && !ctx.IsRoot {
		ctx.Cancel("Authentication required to update pets", 401)
		return nil
	}

	// Business logic - update adoption status based on owner
	if owner, hasOwner := ctx.Data["owner"].(string); hasOwner && strings.TrimSpace(owner) != "" {
		ctx.Data["adoptionStatus"] = "adopted"
		if _, hasAdoptionDate := ctx.Data["adoptionDate"]; !hasAdoptionDate {
			ctx.Data["adoptionDate"] = time.Now().Format(time.RFC3339)
		}

		// Auto-assign microchip if adopted but doesn't have one
		if _, hasMicrochip := ctx.Data["microchipId"]; !hasMicrochip {
			microchipId := fmt.Sprintf("MC%d%s", time.Now().Unix(), generateRandomString(6))
			ctx.Data["microchipId"] = microchipId
		}
	} else {
		ctx.Data["adoptionStatus"] = "available"
		ctx.Data["adoptionDate"] = nil
	}

	// Prevent certain fields from being modified after creation
	if newPetId, hasPetId := ctx.Data["petId"].(string); hasPetId {
		// Get original data to compare (this would need to be implemented based on your system)
		// For now, we'll assume petId shouldn't be changed if it exists in the update
		if originalData := ctx.Query["originalData"]; originalData != nil {
			if originalPetId, hasOriginal := originalData.(map[string]interface{})["petId"].(string); hasOriginal {
				if newPetId != originalPetId {
					ctx.Error("petId", "Pet ID cannot be changed after creation")
				}
			}
		}
	}

	// Validate vet visit date
	if lastVetVisit, hasVetVisit := ctx.Data["lastVetVisit"].(string); hasVetVisit {
		if vetDate, err := time.Parse(time.RFC3339, lastVetVisit); err == nil {
			if vetDate.After(time.Now()) {
				ctx.Error("lastVetVisit", "Vet visit date cannot be in the future")
			}
		}
	}

	// Business logic handled above

	// Hide sensitive fields
	ctx.Hide("createdBy")
	ctx.Hide("updatedBy")

	return nil
}

// Helper function to generate random string (reused from post.go)
func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
