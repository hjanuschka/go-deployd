package main

import "github.com/hjanuschka/go-deployd/internal/events"

// Run filters documents based on user authentication and ownership
func Run(ctx *events.EventContext) error {
    ctx.Log("[get.go] Event triggered", map[string]interface{}{"isRoot": ctx.IsRoot})

    // If the user is root, they can see everything. Do nothing.
    if ctx.IsRoot {
        ctx.Log("[get.go] User is root, skipping filtering.")
        return nil
    }

    // From here, user is NOT root. They must be authenticated.
    if ctx.Me == nil {
        ctx.Log("[get.go] Non-root user is not authenticated. Cancelling.")
        ctx.Cancel("Authentication required", 401)
        return nil
    }

    // Get the current user's ID from the session data.
    var currentUserID string
    if id, ok := ctx.Me["id"].(string); ok {
        currentUserID = id
    }

    if currentUserID == "" {
        ctx.Log("[get.go] Could not determine user ID from session. Cancelling.")
        ctx.Cancel("Unable to determine user ID from session", 500)
        return nil
    }
    ctx.Log("[get.go] Current User ID: %s", currentUserID)

    // If this is a request for a single document, check ownership.
    if docID, exists := ctx.Data["id"]; exists {
        ctx.Log("[get.go] Single document request for ID: %v", docID)
        if ownerID, ok := ctx.Data["userId"].(string); ok {
            if ownerID != currentUserID {
                ctx.Log("[get.go] Ownership check failed. User %s does not own doc owned by %s", currentUserID, ownerID)
                ctx.Cancel("Document not found", 404)
                return nil
            }
        }
    } else {
        // This is a request for multiple documents. Filter the query by the user's ID.
        ctx.Log("[get.go] Multiple document request. Filtering query by userId: %s", currentUserID)
        ctx.Query["userId"] = currentUserID
    }

    return nil
}
