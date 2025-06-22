// Test JavaScript validation event
if (\!this.title || this.title.length < 3) {
  error('title', 'Title must be at least 3 characters');
}

if (this.priority && (this.priority < 1 || this.priority > 5)) {
  error('priority', 'Priority must be between 1 and 5');
}

// Log execution for testing
console.log('JavaScript validation executed for:', this.title);
EOF < /dev/null
