// Test V8 implementation with require() modules
const crypto = require('crypto');
const util = require('util');
const path = require('path');

// Test built-in modules
data.v8Test = {
    cryptoUUID: crypto.randomUUID(),
    cryptoBytes: crypto.randomBytes(8),
    utilIsArray: util.isArray([1, 2, 3]),
    utilIsObject: util.isObject({key: 'value'}),
    pathExt: path.extname('test.js'),
    pathBase: path.basename('/path/to/file.txt')
};

// Test npm module (should return undefined for now)
try {
    const lodash = require('lodash');
    data.v8Test.lodashLoaded = lodash !== undefined;
} catch (e) {
    data.v8Test.lodashError = e.message;
}

// Test console and deployd logging
console.log('V8 test script executed successfully');
deployd.log('V8 engine is working!', {engine: 'v8', modules: 'crypto,util,path'});

data.v8Test.timestamp = new Date().toISOString();
data.v8Test.engine = 'V8';