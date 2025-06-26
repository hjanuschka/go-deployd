package main

// Run filters documents based on user authentication and ownership
func Run(ctx *EventContext) error {
	// Admin users can see all documents
	if ctx.IsRoot {
		return nil
	}

	// Check if user is authenticated via Me object
	if ctx.Me == nil {
		ctx.Cancel("Authentication required", 401)
		return nil
	}

	// Helper function to get user ID from JWT authentication context
	getUserID := func() string {
		// Try to get from Me object
		if ctx.Me != nil {
			if userID, ok := ctx.Me["id"].(string); ok {
				return userID
			}
			if userID, ok := ctx.Me["userId"].(string); ok {
				return userID
			}
			if userID, ok := ctx.Me["UserID"].(string); ok {
				return userID
			}
		}
		return ""
	}

	currentUserID := getUserID()
	if currentUserID == "" {
		ctx.Cancel("Unable to determine user ID", 400)
		return nil
	}

	// For single document requests, check ownership
	if docUserID, exists := ctx.Data["userId"].(string); exists {
		if currentUserID != docUserID {
			ctx.Cancel("Document not found", 404)
			return nil
		}
	} else {
		// Multiple documents - filter by userId
		ctx.Query["userId"] = currentUserID
	}

	return nil
}
