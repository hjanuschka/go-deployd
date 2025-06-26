// Example: JavaScript Event Script with require() Support
// This demonstrates the built-in modules available via require()

// Import available modules
const crypto = require('crypto');
const util = require('util');
const path = require('path');

// Built-in JavaScript features + require() modules
function validateAndEnhanceData() {
    // 1. Crypto utilities for IDs and hashing
    if (!data.trackingId) {
        data.trackingId = 'js_' + crypto.randomUUID().substring(0, 12);
        data.sessionHash = crypto.randomBytes(8).toString('hex');
    }

    // 2. Utility functions for type checking
    if (data.metadata) {
        if (util.isArray(data.metadata)) {
            data.metadataType = 'array';
            data.metadataLength = data.metadata.length;
            data.metadataFirst = data.metadata[0] || null;
        } else if (util.isObject(data.metadata)) {
            data.metadataType = 'object';
            data.metadataKeys = Object.keys(data.metadata);
            data.metadataKeyCount = Object.keys(data.metadata).length;
        } else {
            data.metadataType = typeof data.metadata;
        }
    }

    // 3. Path utilities for file handling
    if (data.attachments && util.isArray(data.attachments)) {
        data.attachmentInfo = data.attachments.map(file => {
            return {
                name: path.basename(file),
                extension: path.extname(file),
                nameWithoutExt: path.basename(file, path.extname(file))
            };
        });
    }

    // 4. Built-in JavaScript Date and Math
    data.createdAt = new Date().toISOString();
    data.createdAtMs = Date.now();
    data.dayOfWeek = new Date().getDay(); // 0-6
    data.month = new Date().getMonth() + 1; // 1-12

    // 5. String processing with built-in methods
    if (data.title) {
        data.titleUpper = data.title.toUpperCase();
        data.titleLower = data.title.toLowerCase();
        data.titleLength = data.title.length;
        data.titleWords = data.title.split(/\s+/).filter(word => word.length > 0);
        data.titleWordCount = data.titleWords.length;
        
        // Generate slug
        data.slug = data.title
            .toLowerCase()
            .replace(/[^a-z0-9\s-]/g, '')
            .replace(/\s+/g, '-')
            .replace(/-+/g, '-')
            .trim('-');
    }

    // 6. Array processing with built-in methods
    if (data.tags && util.isArray(data.tags)) {
        data.tagsUpper = data.tags.map(tag => tag.toUpperCase());
        data.uniqueTags = [...new Set(data.tags)]; // Remove duplicates
        data.tagCount = data.tags.length;
        data.uniqueTagCount = data.uniqueTags.length;
        data.tagsString = data.tags.join(', ');
    }

    // 7. Math operations
    if (data.numbers && util.isArray(data.numbers)) {
        data.numbersSum = data.numbers.reduce((sum, num) => sum + num, 0);
        data.numbersAvg = data.numbersSum / data.numbers.length;
        data.numbersMax = Math.max(...data.numbers);
        data.numbersMin = Math.min(...data.numbers);
    }

    // 8. Priority scoring with external random
    if (data.priority) {
        const randomFactor = Math.random();
        data.priorityScore = data.priority * 10 + Math.round(randomFactor * 5);
        data.priorityLevel = data.priority >= 5 ? 'high' : 
                           data.priority >= 3 ? 'medium' : 'low';
    }

    // 9. User context integration
    if (me) {
        data.createdBy = me.id || me.username;
        data.createdByName = me.name || me.username;
        data.userRole = me.role || 'user';
    }

    // 10. Complex validation with custom logic
    if (data.email) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(data.email)) {
            error('email', 'Invalid email format');
        } else {
            data.emailDomain = data.email.split('@')[1];
            data.emailLocalPart = data.email.split('@')[0];
        }
    }

    // JSON processing
    if (data.jsonString && typeof data.jsonString === 'string') {
        try {
            data.parsedJson = JSON.parse(data.jsonString);
            data.jsonValid = true;
        } catch (e) {
            data.jsonValid = false;
            error('jsonString', 'Invalid JSON format');
        }
    }
}

// Execute the main function
validateAndEnhanceData();

/*
Available require() modules:

1. require('crypto'):
   - randomUUID(): Generate UUID-like strings
   - randomBytes(size): Generate random byte arrays

2. require('util'):
   - isArray(obj): Check if object is array
   - isObject(obj): Check if object is object

3. require('path'):
   - extname(path): Get file extension
   - basename(path): Get filename from path

Built-in JavaScript features available:
- All standard JavaScript: Array, Object, String, Number, Date, Math, JSON
- Regular expressions and pattern matching
- Functional programming: map, filter, reduce, forEach
- ES6+ features: destructuring, spread operator, template literals

Usage in config.json:
{
  "eventConfig": {
    "validate": {
      "runtime": "js"
    }
  }
}

Test example:
curl -X POST http://localhost:2405/todos \
  -H "Content-Type: application/json" \
  -d '{
    "title": "JavaScript External Libraries Test",
    "metadata": {"type": "test", "version": 2},
    "tags": ["javascript", "test", "require"],
    "attachments": ["document.pdf", "image.jpg"],
    "numbers": [1, 2, 3, 4, 5],
    "priority": 4,
    "email": "test@example.com",
    "jsonString": "{\"nested\": \"data\"}"
  }'

This will add many computed fields using built-in JS + require() modules.
*/