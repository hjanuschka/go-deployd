// Example PUT event script for todos collection (JavaScript version)

// Track what changed
var changed = [];
if (previous) {
    Object.keys(data).forEach(function(key) {
        if (data[key] !== previous[key]) {
            changed.push(key);
        }
    });
}

// Don't allow changing the creation date
if (previous && data.createdAt !== previous.createdAt) {
    error('createdAt', 'Cannot modify creation date');
}

// Update the lastModified timestamp
data.updatedAt = new Date().toISOString();

// If status changed to completed, set completedAt
if (changed.includes('completed') && data.completed === true) {
    data.completedAt = new Date().toISOString();
} else if (changed.includes('completed') && data.completed === false) {
    delete data.completedAt;
}

// Track who made the change
if (me) {
    data.updatedBy = me.id;
}

// Validate that completed todos have a title
if (data.completed && !data.title) {
    error('title', 'Completed todos must have a title');
}

// Example: Emit event when todo is completed
if (changed.includes('completed') && data.completed) {
    emit('todoCompleted', {
        id: data.id,
        title: data.title,
        completedBy: me ? me.id : null
    });
}