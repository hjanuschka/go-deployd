// JavaScript put event for testing data modification with V8 and npm modules
// const moment = require('moment'); // Disabled for now
const _ = require('lodash');

console.log('JS Put: Updating record with V8 engine');

// Add update metadata
data.updatedBy = 'js-event-system';
data.updatedAt = new Date().toISOString();
data.version = _.get(data, 'version', 1) + 1;

// Track what fields were modified using lodash
const originalFields = ['title', 'description', 'priority', 'status'];
const modifiedFields = _.intersection(Object.keys(data), originalFields);

data.modifiedFields = modifiedFields;
data.modificationCount = modifiedFields.length;
data.hasModifications = modifiedFields.length > 0;

// Test lodash data transformation
if (data.title) {
    data.titleHistory = _.get(data, 'titleHistory', []);
    data.titleHistory.push({
        value: data.title,
        modifiedAt: new Date().toISOString(),
        version: data.version
    });
}

// Update status with smart defaults
data.status = data.status || 'updated';

// Test moment date calculations
data.updatedTimestamp = Math.floor(Date.now() / 1000);
data.lastModified = 'just now';
data.updateDate = new Date().toISOString().split('T')[0];

// Test lodash array manipulation
if (data.tags && _.isArray(data.tags)) {
    data.tags = _.uniq([...data.tags, 'js-updated']);
    data.uniqueTagCount = _.uniq(data.tags).length;
}

// Test object merging with lodash
data.updateMetadata = _.merge(_.get(data, 'updateMetadata', {}), {
    engine: 'v8',
    runtime: 'javascript',
    npmModules: ['lodash', 'moment'],
    updateCount: _.get(data, 'updateMetadata.updateCount', 0) + 1
});

// Test conditional logic
if (data.priority) {
    const priorityChanges = _.get(data, 'priorityChanges', []);
    priorityChanges.push({
        newValue: data.priority,
        changedAt: new Date().toISOString(),
        version: data.version
    });
    data.priorityChanges = priorityChanges;
}

console.log('JS Put: Record update completed, version', data.version);
deployd.log('JavaScript put event completed', {
    version: data.version,
    modificationsCount: data.modificationCount,
    engine: 'v8'
});