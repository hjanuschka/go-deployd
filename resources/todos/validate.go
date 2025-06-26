package main

// On Validate - Validate data before saving (Go version)
func Run(ctx *EventContext) error {
	// Validate required fields
	title, hasTitle := ctx.Data["title"].(string)
	if !hasTitle || title == "" {
		ctx.Error("title", "Title is required")
	}

	// Custom validation
	if priority, ok := ctx.Data["priority"].(float64); ok {
		if priority < 1 || priority > 5 {
			ctx.Error("priority", "Priority must be between 1 and 5")
		}
	}

	return nil
}
