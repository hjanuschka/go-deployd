// Simple validation for todo items
if (!this.title || this.title.length < 1) {
    cancel("Title is required", 400);
}

if (this.title.length > 200) {
    cancel("Title is too long (max 200 characters)", 400);
}

if (this.priority !== undefined && (this.priority < 1 || this.priority > 5)) {
    cancel("Priority must be between 1 and 5", 400);
}