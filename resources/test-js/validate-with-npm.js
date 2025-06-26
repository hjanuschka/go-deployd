// JavaScript validation event for testing data modification with V8 and npm modules
const crypto = require('crypto');
const util = require('util');
// const moment = require('moment'); // Disabled for now
const _ = require('lodash');

console.log('JS Validation: Starting validation with V8 engine');

// Basic validation
if (!data.title || data.title.trim() === '') {
    error('title', 'Title is required');
}

// Test data modification - add computed fields using npm modules
data.validated = true;
data.validatedAt = new Date().toISOString();
data.validationId = crypto.randomUUID();

// Test lodash string manipulation
if (data.title) {
    data.titleUpper = _.toUpper(data.title);
    data.titleLength = data.title.length;
    data.slug = _.kebabCase(data.title);
    data.titleWords = _.words(data.title);
    data.titleCapitalized = _.capitalize(data.title);
}

// Test util type checking
if (data.metadata) {
    data.metadataType = util.isArray(data.metadata) ? 'array' : 
                       util.isObject(data.metadata) ? 'object' : 
                       typeof data.metadata;
}

// Test conditional logic with lodash
if (data.priority) {
    data.priorityLevel = data.priority >= 5 ? 'high' : 
                        data.priority >= 3 ? 'medium' : 'low';
    data.priorityScore = _.random(1, 10) * data.priority;
}

// Test array/object modification with lodash
if (!data.tags) {
    data.tags = ['js-validated', 'v8-engine'];
} else if (_.isArray(data.tags)) {
    data.tags = _.uniq([...data.tags, 'js-validated']);
    data.tagCount = data.tags.length;
}

if (!data.metadata) {
    data.metadata = {
        source: 'js-validation',
        engine: 'v8',
        version: '2.0',
        npmModules: ['lodash', 'crypto']
    };
} else if (_.isObject(data.metadata)) {
    data.metadata = _.merge(data.metadata, {
        processedBy: 'v8-validation',
        processedAt: new Date().toISOString()
    });
}

// Test basic date manipulation (without moment for now)
data.validationTimestamp = Math.floor(Date.now() / 1000);
data.expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
data.dayOfWeek = new Date().toLocaleDateString('en-US', { weekday: 'long' });

// Test crypto functions
data.securityHash = crypto.randomBytes(16).toString('hex');
data.sessionId = 'js_' + crypto.randomUUID().substring(0, 8);

console.log('JS Validation: Completed validation, added', Object.keys(data).length, 'fields');
deployd.log('JavaScript validation completed successfully', {
    engine: 'v8',
    modules: ['lodash', 'crypto'],
    fieldsAdded: Object.keys(data).filter(k => !['title', 'description', 'priority'].includes(k)).length
});