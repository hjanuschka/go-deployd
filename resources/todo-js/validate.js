// Run function for todo validation using new context pattern
function Run(context) {
    context.log("Todo validation executing with Run(context) pattern");
    
    // Validate title
    if (!context.data.title || context.data.title.length < 1) {
        context.cancel("Title is required", 400);
        return;
    }
    
    if (context.data.title.length > 200) {
        context.cancel("Title is too long (max 200 characters)", 400);
        return;
    }
    
    // Validate priority
    if (context.data.priority !== undefined && (context.data.priority < 1 || context.data.priority > 5)) {
        context.cancel("Priority must be between 1 and 5", 400);
        return;
    }
    
    context.log("Todo validation passed successfully");
}