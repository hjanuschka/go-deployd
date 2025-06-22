// Example PUT event script for todos collection (JavaScript version)

// Track what changed
var changed = [];
if (previous) {
    Object.keys(this).forEach(function(key) {
        if (this[key] !== previous[key]) {
            changed.push(key);
        }
    }.bind(this));
}

// Don't allow changing the creation date
if (previous && this.createdAt !== previous.createdAt) {
    error('createdAt', 'Cannot modify creation date');
}

// Update the lastModified timestamp
this.updatedAt = new Date().toISOString();

// If status changed to completed, set completedAt
if (changed.includes('completed') && this.completed === true) {
    this.completedAt = new Date().toISOString();
} else if (changed.includes('completed') && this.completed === false) {
    delete this.completedAt;
}

// Track who made the change
if (me) {
    this.updatedBy = me.id;
}

// Validate that completed todos have a title
if (this.completed && !this.title) {
    error('title', 'Completed todos must have a title');
}

// Example: Emit event when todo is completed
if (changed.includes('completed') && this.completed) {
    emit('todoCompleted', {
        id: this.id,
        title: this.title,
        completedBy: me ? me.id : null
    });
}