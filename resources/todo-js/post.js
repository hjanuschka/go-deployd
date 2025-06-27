// Simple post-processing for newly created todos
console.log("Created new todo:", this.title);

// Set default values if not provided
if (this.completed === undefined) {
    this.completed = false;
}

if (this.priority === undefined) {
    this.priority = 1;
}