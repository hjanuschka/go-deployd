// Simple JavaScript post event without npm modules
console.log('Simple JS Post: Creating new record');

// Add creation metadata
data.createdBy = 'js-event-system';
data.createdAt = new Date().toISOString();
data.id = 'js_' + Math.random().toString(36).substr(2, 9);

// Set default status if not provided
if (!data.status) {
    data.status = 'created';
}

// Test basic computations
data.createdTimestamp = Math.floor(Date.now() / 1000);
data.createdDate = new Date().toISOString().split('T')[0];
data.createdTime = new Date().toISOString().split('T')[1].split('.')[0];

// Test metadata
data.metadata = {
    source: 'js-post-simple',
    engine: 'v8',
    version: '1.0',
    timestamp: Date.now()
};

console.log('Simple JS Post: Completed post processing');