// BeforeRequest event - runs before all other events
// Available variables: event (GET, POST, PUT, DELETE), ctx, query, me

console.log("BeforeRequest event triggered for:", event);

// Example: Require authentication for all requests except GET
if (event !== "GET" && !me) {
    cancel("Authentication required", 401);
}

// Example: Modify query parameters
if (event === "GET") {
    // Add default sorting if not specified
    if (!query.$sort) {
        query.$sort = { createdAt: -1 };
    }
    
    // Limit query results for non-admin users
    if (!me || me.role !== "admin") {
        query.$limit = Math.min(query.$limit || 50, 50);
    }
}

// Example: Log all requests for auditing
deployd.log("Request audit", {
    event: event,
    user: me ? me.username : "anonymous",
    query: query,
    timestamp: new Date()
});