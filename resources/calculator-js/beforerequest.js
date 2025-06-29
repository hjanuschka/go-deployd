// BeforeRequest for calculator-js
console.log('Calculator JS BeforeRequest called');

if (typeof setHeader !== 'undefined') {
  setHeader("X-Calculator", "JS");
}