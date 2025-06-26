package main

// Run filters or modifies retrieved documents  
func Run(ctx *EventContext) error {
    // Hide sensitive fields (syntax sugar for delete)
    ctx.Hide("password")
    ctx.Hide("verificationToken")
    
    // Log user access for demo
    if userID, ok := ctx.Data["id"].(string); ok {
        if email, ok := ctx.Data["email"].(string); ok {
            ctx.Log("Loaded User ID: " + userID + " with email " + email)
        }
    }
    
    return nil
}