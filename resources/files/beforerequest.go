// Run performs early validation before file upload processing
func Run(ctx *EventContext) error {
	// This event runs before any file processing, allowing for early rejection
	// based on request method and user authentication
	
	// Check if this is a file upload request
	if ctx.Method != "POST" {
		return nil // Only validate POST requests (uploads)
	}
	
	// Require authentication (either user auth or master key)
	if (ctx.Me == nil || ctx.Me["id"] == nil) && !ctx.IsRoot {
		ctx.Cancel("Authentication required for file uploads", 401)
		return nil
	}
	
	// Example: Check user permissions or quotas
	// This is useful for early rejection before processing large files
	if !ctx.IsRoot {
		// Non-admin users have restrictions
		logData := map[string]interface{}{
			"method": ctx.Method,
		}
		if ctx.Me != nil && ctx.Me["id"] != nil {
			logData["userId"] = ctx.Me["id"]
		}
		ctx.Log("File upload attempt by regular user", logData)
	}
	
	// Example: Rate limiting based on user
	// You could implement rate limiting logic here
	
	// Log based on authentication method
	logData := map[string]interface{}{
		"method": ctx.Method,
		"isRoot": ctx.IsRoot,
	}
	
	if ctx.Me != nil && ctx.Me["id"] != nil {
		logData["userId"] = ctx.Me["id"]
	}
	
	ctx.Log("File upload request validated", logData)
	
	return nil
}