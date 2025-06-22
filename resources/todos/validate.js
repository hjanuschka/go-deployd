if (!this.title || this.title.trim() === '') {
  error('title', 'Title is required');
}
if (this.title && this.title.length < 3) {
  error('title', 'Title must be at least 3 characters');
}