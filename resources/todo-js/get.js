// Simple get processing for todos - back to this.* pattern for testing
// Add a computed field showing completion status
this.status = this.completed ? "Done" : "Pending";

// Format the creation date for display
if (this.createdAt) {
    this.formattedDate = new Date(this.createdAt).toLocaleDateString();
}

// Add metadata about the processing
this.processedBy = "JavaScript this.* pattern";
this.processedAt = new Date().toISOString();