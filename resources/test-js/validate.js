// Simple JavaScript validation without npm modules
console.log('Simple JS Validation: Starting validation');

// Basic validation
if (!data.title || data.title.trim() === '') {
    error('title', 'Title is required');
}

// Test data modification - add computed fields
data.validated = true;
data.validatedAt = new Date().toISOString();
data.validationId = 'js_' + Math.random().toString(36).substr(2, 9);

// Test string manipulation
if (data.title) {
    data.titleUpper = data.title.toUpperCase();
    data.titleLength = data.title.length;
    data.slug = data.title.toLowerCase().replace(/\s+/g, '-');
}

// Test conditional logic
if (data.priority) {
    data.priorityLevel = data.priority >= 5 ? 'high' : 
                        data.priority >= 3 ? 'medium' : 'low';
    data.priorityScore = Math.floor(Math.random() * 10) * data.priority;
}

// Test array/object modification
if (!data.tags) {
    data.tags = ['js-validated', 'simple-validation'];
}

data.metadata = {
    source: 'js-validation-simple',
    engine: 'v8',
    version: '1.0',
    timestamp: Date.now()
};

console.log('Simple JS Validation: Completed validation, modified data');