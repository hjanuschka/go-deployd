// Unified Run(context) pattern for calculator JavaScript implementation
function Run(context) {
    context.log("Calculator GET event executing with unified Run(context) pattern");
    
    const parts = context.data.parts;
    
    if (parts && parts.length > 0 && parts[0] === "add" && parts.length === 3) {
        const a = parseInt(parts[1]);
        const b = parseInt(parts[2]);
        
        context.log("Calculator computing: " + a + " + " + b);
        
        context.data.operation = "add";
        context.data.operands = [a, b];
        context.data.result = a + b;
        context.data.context_type = "noStore_js_unified";
    } else {
        context.data.error = "Use: /calculator-js/add/num1/num2";
        context.data.usage = "Try: /calculator-js/add/10/5";
    }
}