import "fmt"

// Run controls file access and listing
func Run(ctx *EventContext) error {
	// For single file access (GET /files/{id})
	if ctx.Data != nil && ctx.Data["id"] != nil {
		// Example: Only allow file owner or admin to access (uncomment to enable)
		// if !ctx.IsRoot && ctx.Data["uploadedBy"] != ctx.Me["id"] {
		// 	ctx.Cancel("You can only access your own files", 403)
		// 	return nil
		// }

		// Add access logging
		ctx.Log("File accessed", map[string]interface{}{
			"id":         ctx.Data["id"],
			"name":       ctx.Data["originalName"],
			"accessedBy": ctx.Me["id"],
		})
	}

	// For file listing (GET /files)
	if ctx.Data == nil || ctx.Data["id"] == nil {
		// Require authentication for listing files
		if ctx.Me == nil || ctx.Me["id"] == nil {
			ctx.Cancel("Authentication required to list files", 401)
		return nil
		}

		// The files resource will automatically filter by user
		// unless the user is root/admin
		ctx.Log(fmt.Sprintf("Files listed by user: %v", ctx.Me["id"]))
	}

	return nil
}