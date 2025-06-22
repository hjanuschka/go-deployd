// Test file showing how to use npm modules in go-deployd events
const _ = require('lodash');
const moment = require('moment');
const { v4: uuidv4 } = require('uuid');
const validator = require('validator');

console.log('Testing npm modules:');
console.log('Lodash:', _.capitalize('hello world'));
console.log('Moment:', moment().format('YYYY-MM-DD'));
console.log('UUID:', uuidv4());
console.log('Validator:', validator.isEmail('test@example.com'));
