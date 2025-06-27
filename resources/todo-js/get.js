// Simple get processing for todos
// Add a computed field showing completion status
this.status = this.completed ? "Done" : "Pending";

// Format the creation date for display
if (this.createdAt) {
    this.formattedDate = new Date(this.createdAt).toLocaleDateString();
}