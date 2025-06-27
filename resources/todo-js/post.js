// Simple post-processing for newly created todos
// Set default values if not provided
if (this.completed === undefined) {
    this.completed = false;
}

if (this.priority === undefined) {
    this.priority = 1;
}

// Log todo creation using proper logging
ctx.Log("Todo created successfully", {
    title: this.title,
    action: "post",
    completed: this.completed,
    priority: this.priority
});