// Unified Run(context) pattern for todo GET events
function Run(context) {
    context.log("Todo GET event executing with unified Run(context) pattern");
    
    // Add a computed field showing completion status
    context.data.status = context.data.completed ? "Done" : "Pending";
    
    // Format the creation date for display
    if (context.data.createdAt) {
        context.data.formattedDate = new Date(context.data.createdAt).toLocaleDateString();
    }
    
    // Add metadata about the processing
    context.data.processedBy = "JavaScript Run(context) unified";
    context.data.processedAt = new Date().toISOString();
}