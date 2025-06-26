// JavaScript post event for testing data modification with V8 and npm modules
// const moment = require('moment'); // Disabled for now
const { v4: uuidv4 } = require('uuid');
const _ = require('lodash');

console.log('JS Post: Creating new record with V8 engine');

// Add creation metadata
data.createdBy = 'js-event-system';
data.createdAt = new Date().toISOString();
data.id = uuidv4();

// Test user context
if (me) {
    data.createdByUser = me.id || me.username;
    data.userRole = me.role || 'user';
    data.userName = me.name || 'Unknown';
}

// Test isRoot functionality
if (isRoot) {
    data.createdByAdmin = true;
    data.adminPrivileges = ['read', 'write', 'delete', 'admin'];
    data.systemAccess = true;
}

// Set default status with lodash
data.status = _.defaultTo(data.status, 'created');

// Test query parameters
if (query && Object.keys(query).length > 0) {
    data.queryParams = query;
    data.hasQueryParams = true;
}

// Test moment date calculations
data.createdTimestamp = Math.floor(Date.now() / 1000);
data.createdDate = new Date().toISOString().split('T')[0];
data.createdTime = new Date().toISOString().split('T')[1].split('.')[0];

// Test lodash object manipulation
data.summary = {
    recordType: 'test-js',
    engine: 'v8',
    npmModules: ['lodash', 'moment', 'uuid'],
    features: ['data-modification', 'npm-support', 'user-context']
};

// Test array operations
data.processingSteps = [
    'validation-passed',
    'post-event-triggered',
    'data-enhanced',
    'ready-for-storage'
];

console.log('JS Post: Record creation completed');
deployd.log('JavaScript post event completed', {
    recordId: data.id,
    engine: 'v8',
    createdBy: data.createdBy
});