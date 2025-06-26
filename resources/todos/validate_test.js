// Test JavaScript event capabilities with require() support
const crypto = require('crypto');
const util = require('util');
const path = require('path');

// Basic validation
if (!data.title || data.title.trim() === '') {
  error('title', 'Title is required');
}

if (data.title && data.title.length < 3) {
  error('title', 'Title must be at least 3 characters');
}

// Use require('crypto') for UUID generation
if (!data.uniqueId) {
  data.uniqueId = crypto.randomUUID();
  data.trackingId = 'js_' + data.uniqueId.substring(0, 8);
}

// Use require('util') for type checking
if (data.metadata) {
  if (util.isArray(data.metadata)) {
    data.metadataType = 'array';
    data.metadataCount = data.metadata.length;
  } else if (util.isObject(data.metadata)) {
    data.metadataType = 'object';
    data.metadataKeys = Object.keys(data.metadata);
  } else {
    data.metadataType = typeof data.metadata;
  }
}

// Use require('path') for filename operations
if (data.filename) {
  data.fileExtension = path.extname(data.filename);
  data.fileName = path.basename(data.filename);
}

// Test array operations with built-in methods
if (data.tags && Array.isArray(data.tags)) {
  data.tagCount = data.tags.length;
  data.uppercaseTags = data.tags.map(tag => tag.toUpperCase());
  data.tagHash = crypto.randomBytes(4).toString('hex');
}

// Email validation
if (data.email) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(data.email)) {
    error('email', 'Invalid email format');
  }
}