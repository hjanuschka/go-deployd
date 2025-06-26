package main

// Run filters or modifies retrieved documents
func Run(ctx *EventContext) error {
	ctx.Data["heli"] = 1234
	// Hide sensitive fields from non-admin users
	if !ctx.IsRoot {
		ctx.Hide("internalNotes")
		ctx.Hide("cost")
	}

	// Only show user's own documents
	if !ctx.IsRoot && ctx.Me != nil {
		if userId, _ := ctx.Data["userId"].(string); userId != ctx.Me["id"] {
			ctx.Cancel("Not authorized", 403)
		}
	}

	return nil
}
