// Run performs early validation before file upload processing
func Run(ctx *EventContext) error {
	// This event runs before any file processing, allowing for early rejection
	// based on request method and user authentication
	
	// Check if this is a file upload request
	if ctx.Method != "POST" {
		return nil // Only validate POST requests (uploads)
	}
	
	// Example: Require authentication before processing
	if ctx.Me == nil || ctx.Me["id"] == nil {
		ctx.Cancel("Authentication required for file uploads", 401)
		return nil
	}
	
	// Example: Check user permissions or quotas
	// This is useful for early rejection before processing large files
	if !ctx.IsRoot {
		// Non-admin users have restrictions
		ctx.Log("File upload attempt by regular user", map[string]interface{}{
			"userId": ctx.Me["id"],
			"method": ctx.Method,
		})
	}
	
	// Example: Rate limiting based on user
	// You could implement rate limiting logic here
	
	ctx.Log("File upload request validated", map[string]interface{}{
		"userId": ctx.Me["id"],
		"method": ctx.Method,
	})
	
	return nil
}