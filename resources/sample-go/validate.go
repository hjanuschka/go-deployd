package main

import (
	"strings"
)

// Validate pet data before saving (Go version)
func Run(ctx *EventContext) error {
	// Validate required fields
	name, hasName := ctx.Data["name"].(string)
	if !hasName || name == "" {
		ctx.Error("name", "Pet name is required")
	}

	species, hasSpecies := ctx.Data["species"].(string)
	if !hasSpecies || species == "" {
		ctx.Error("species", "Pet species is required")
	}

	breed, hasBreed := ctx.Data["breed"].(string)
	if !hasBreed || breed == "" {
		ctx.Error("breed", "Pet breed is required")
	}

	age, hasAge := ctx.Data["age"].(float64)
	if !hasAge || age < 0 {
		ctx.Error("age", "Pet age must be a positive number")
	}

	// Validate optional fields
	if weight, ok := ctx.Data["weight"].(float64); ok && weight <= 0 {
		ctx.Error("weight", "Weight must be greater than 0")
	}

	if age > 50 {
		ctx.Error("age", "Pet age seems unrealistic (max 50 years)")
	}

	// Validate species-specific constraints
	if species == "dog" && age > 25 {
		ctx.Error("age", "Dog age seems unrealistic (max 25 years)")
	}

	if species == "cat" && age > 20 {
		ctx.Error("age", "Cat age seems unrealistic (max 20 years)")
	}

	// Normalize data
	if hasName {
		ctx.Data["name"] = strings.TrimSpace(name)
	}
	if hasSpecies {
		ctx.Data["species"] = strings.ToLower(strings.TrimSpace(species))
	}
	if hasBreed {
		ctx.Data["breed"] = strings.TrimSpace(breed)
	}

	// Color normalization
	if color, ok := ctx.Data["color"].(string); ok {
		ctx.Data["color"] = strings.ToLower(strings.TrimSpace(color))
	}

	// Microchip ID validation
	if microchipId, ok := ctx.Data["microchipId"].(string); ok && microchipId != "" {
		if len(microchipId) < 10 || len(microchipId) > 15 {
			ctx.Error("microchipId", "Microchip ID must be between 10 and 15 characters")
		}
	}

	// Access control - only authenticated users can create pets
	if ctx.Me == nil {
		ctx.Cancel("Authentication required to manage pets", 401)
		return nil
	}

	return nil
}
