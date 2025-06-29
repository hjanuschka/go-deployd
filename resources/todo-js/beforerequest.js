// Run function for todo BeforeRequest events using new context pattern
function Run(context) {
    context.log("Todo BeforeRequest event executing with Run(context) pattern");
    
    // Note: BeforeRequest events have different context structure
    // Access request method, query, and user through context
    const event = context.data.method || "GET";
    const query = context.data.query || {};
    const me = context.data.user || null;
    
    context.log("BeforeRequest event triggered for: " + event);
    
    // Example: Require authentication for all requests except GET
    if (event !== "GET" && !me) {
        context.cancel("Authentication required", 401);
        return;
    }
    
    // Example: Modify query parameters for GET requests
    if (event === "GET") {
        // Add default sorting if not specified
        if (!query.$sort) {
            query.$sort = { createdAt: -1 };
        }
        
        // Limit query results for non-admin users
        if (!me || me.role !== "admin") {
            query.$limit = Math.min(query.$limit || 50, 50);
        }
        
        // Update the query in context
        context.data.query = query;
    }
    
    // Log all requests for auditing
    context.log("Request audit", {
        event: event,
        user: me ? me.username : "anonymous",
        query: query,
        timestamp: new Date()
    });
}