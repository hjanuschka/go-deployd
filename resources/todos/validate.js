// Test npm module loading with V8 engine
console.log('Testing npm modules with V8...');

// Test lodash (utility library)
try {
    const _ = require('lodash');
    data.lodashTest = {
        loaded: true,
        capitalize: _.capitalize('hello world'),
        chunk: _.chunk([1, 2, 3, 4, 5], 2),
        random: _.random(1, 100)
    };
    console.log('✅ Lodash loaded successfully');
} catch (e) {
    data.lodashTest = { loaded: false, error: e.message };
    console.log('❌ Lodash failed:', e.message);
}

// Test moment (date library)
try {
    const moment = require('moment');
    data.momentTest = {
        loaded: true,
        now: moment().format('YYYY-MM-DD HH:mm:ss'),
        relative: moment().fromNow(),
        future: moment().add(7, 'days').format('YYYY-MM-DD')
    };
    console.log('✅ Moment loaded successfully');
} catch (e) {
    data.momentTest = { loaded: false, error: e.message };
    console.log('❌ Moment failed:', e.message);
}

// Test uuid (UUID generation)
try {
    const { v4: uuidv4, v1: uuidv1 } = require('uuid');
    data.uuidTest = {
        loaded: true,
        v4: uuidv4(),
        v1: uuidv1()
    };
    console.log('✅ UUID loaded successfully');
} catch (e) {
    data.uuidTest = { loaded: false, error: e.message };
    console.log('❌ UUID failed:', e.message);
}

// Test validator (string validation)
try {
    const validator = require('validator');
    data.validatorTest = {
        loaded: true,
        isEmail: validator.isEmail('test@example.com'),
        isURL: validator.isURL('https://example.com'),
        isJSON: validator.isJSON('{"valid": true}')
    };
    console.log('✅ Validator loaded successfully');
} catch (e) {
    data.validatorTest = { loaded: false, error: e.message };
    console.log('❌ Validator failed:', e.message);
}

// Basic validation (existing)
if (!data.title || data.title.trim() === '') {
    error('title', 'Title is required');
}

// Enhanced with npm modules
data.npmTestCompleted = true;
data.npmTestTimestamp = new Date().toISOString();

deployd.log('NPM module test completed', {
    lodash: data.lodashTest?.loaded,
    moment: data.momentTest?.loaded,
    uuid: data.uuidTest?.loaded,
    validator: data.validatorTest?.loaded
});