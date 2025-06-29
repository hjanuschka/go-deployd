// Run function for todo POST events using new context pattern
function Run(context) {
    context.log("Todo POST event executing with Run(context) pattern");
    
    // Set default values if not provided
    if (context.data.completed === undefined) {
        context.data.completed = false;
    }
    
    if (context.data.priority === undefined) {
        context.data.priority = 1;
    }
    
    // Add creation metadata
    context.data.createdBy = "JavaScript Run(context)";
    
    // Log todo creation using proper logging
    context.log("Todo created successfully", {
        title: context.data.title,
        action: "post", 
        completed: context.data.completed,
        priority: context.data.priority
    });
}