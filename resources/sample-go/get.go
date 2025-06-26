package main

import (
	"strings"
	"time"
)

// Process data when retrieving pets (Go version)
func Run(ctx *EventContext) error {
	// Hide sensitive information from non-authenticated users
	if ctx.Me == nil && !ctx.IsRoot {
		ctx.Hide("createdBy")
		ctx.Hide("updatedBy")
		ctx.Hide("vetNotes")         // Private vet notes only for authenticated users
		ctx.Hide("emergencyContact") // Emergency contact info is sensitive
		ctx.Hide("microchipId")      // Microchip ID could be sensitive
	}

	// Add computed fields
	if age, hasAge := ctx.Data["age"].(float64); hasAge {
		var ageCategory string
		switch {
		case age < 1:
			ageCategory = "baby"
		case age < 3:
			ageCategory = "young"
		case age < 8:
			ageCategory = "adult"
		default:
			ageCategory = "senior"
		}
		ctx.Data["ageCategory"] = ageCategory
	}

	// Add availability status
	owner, hasOwner := ctx.Data["owner"].(string)
	ctx.Data["isAvailableForAdoption"] = !hasOwner || strings.TrimSpace(owner) == ""

	// Calculate days since last vet visit
	if lastVetVisit, hasVetVisit := ctx.Data["lastVetVisit"].(string); hasVetVisit {
		if vetDate, err := time.Parse(time.RFC3339, lastVetVisit); err == nil {
			daysSinceVet := int(time.Since(vetDate).Hours() / 24)
			ctx.Data["daysSinceLastVetVisit"] = daysSinceVet

			// Add health status indicator
			switch {
			case daysSinceVet > 365:
				ctx.Data["vetStatus"] = "overdue"
			case daysSinceVet > 300:
				ctx.Data["vetStatus"] = "due_soon"
			default:
				ctx.Data["vetStatus"] = "current"
			}
		}
	}

	// Format dates for display
	if adoptionDate, hasAdoptionDate := ctx.Data["adoptionDate"].(string); hasAdoptionDate {
		if date, err := time.Parse(time.RFC3339, adoptionDate); err == nil {
			ctx.Data["adoptionDateFormatted"] = date.Format("2006-01-02")
		}
	}

	if createdAt, hasCreatedAt := ctx.Data["createdAt"].(string); hasCreatedAt {
		if date, err := time.Parse(time.RFC3339, createdAt); err == nil {
			ctx.Data["createdAtFormatted"] = date.Format("2006-01-02")
		}
	}

	// Add species-specific information
	if species, hasSpecies := ctx.Data["species"].(string); hasSpecies {
		switch strings.ToLower(species) {
		case "dog":
			ctx.Data["exerciseNeeds"] = "daily walks required"
			ctx.Data["socialNeeds"] = "pack animal - enjoys company"
		case "cat":
			ctx.Data["exerciseNeeds"] = "indoor play and climbing"
			ctx.Data["socialNeeds"] = "independent but enjoys attention"
		case "bird":
			ctx.Data["exerciseNeeds"] = "flight time outside cage"
			ctx.Data["socialNeeds"] = "social interaction important"
		}
	}

	// Access logged for audit trail
	deployd.Log("Pet data accessed", map[string]interface{}{
		"petId":        ctx.Data["id"],
		"user":         ctx.Me,
		"hiddenFields": len(ctx.Me) == 0,
	})

	return nil
}
