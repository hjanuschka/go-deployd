import (
	"fmt"
	"strconv"
)

// Run handles GET requests for calculator-go (noStore collection)
func Run(ctx *EventContext) error {
	// Check if we have JSON body data (operation, num1, num2)
	if operation, hasOp := ctx.Data["operation"].(string); hasOp {
		if num1Val, hasNum1 := ctx.Data["num1"]; hasNum1 {
			if num2Val, hasNum2 := ctx.Data["num2"]; hasNum2 {
				// Parse numbers from JSON body
				var num1, num2 float64
				var err error
				
				switch v := num1Val.(type) {
				case float64:
					num1 = v
				case int:
					num1 = float64(v)
				case string:
					num1, err = strconv.ParseFloat(v, 64)
					if err != nil {
						ctx.Cancel("Invalid num1 in JSON body", 400)
						return nil
					}
				default:
					ctx.Cancel("num1 must be a number", 400)
					return nil
				}
				
				switch v := num2Val.(type) {
				case float64:
					num2 = v
				case int:
					num2 = float64(v)
				case string:
					num2, err = strconv.ParseFloat(v, 64)
					if err != nil {
						ctx.Cancel("Invalid num2 in JSON body", 400)
						return nil
					}
				default:
					ctx.Cancel("num2 must be a number", 400)
					return nil
				}
				
				// Perform calculation
				result, err := performCalculation(operation, num1, num2)
				if err != nil {
					ctx.Cancel(err.Error(), 400)
					return nil
				}
				
				ctx.Data["operation"] = operation
				ctx.Data["operands"] = []float64{num1, num2}
				ctx.Data["result"] = result
				ctx.Data["input_method"] = "json_body"
				ctx.Data["context_type"] = "noStore_go"
				
				return nil
			}
		}
	}
	
	// URL-based operation: /calculator-go/add/5/3
	partsInterface, hasParts := ctx.Data["parts"]
	
	if !hasParts {
		ctx.Data["message"] = "Calculator Go API"
		ctx.Data["usage"] = "GET /calculator-go/add/5/3 or POST JSON {operation: 'add', num1: 5, num2: 3}"
		return nil
	}
	
	// Convert parts to string slice
	var parts []string
	if partsSlice, ok := partsInterface.([]string); ok {
		parts = partsSlice
	} else if partsSliceInterface, ok := partsInterface.([]interface{}); ok {
		for _, part := range partsSliceInterface {
			if str, ok := part.(string); ok {
				parts = append(parts, str)
			}
		}
	}
	
	// Basic calculator logic
	if len(parts) == 0 {
		ctx.Data["message"] = "Calculator Go API"
		ctx.Data["usage"] = "GET /calculator-go/add/5/3 or POST JSON {operation: 'add', num1: 5, num2: 3}"
		return nil
	}
	
	if len(parts) != 3 {
		ctx.Cancel("Use /calculator-go/operation/num1/num2 or POST JSON {operation, num1, num2}", 400)
		return nil
	}
	
	operation := parts[0]
	num1, err1 := strconv.ParseFloat(parts[1], 64)
	num2, err2 := strconv.ParseFloat(parts[2], 64)
	
	if err1 != nil || err2 != nil {
		ctx.Cancel("Invalid numbers in URL", 400)
		return nil
	}
	
	result, err := performCalculation(operation, num1, num2)
	if err != nil {
		ctx.Cancel(err.Error(), 400)
		return nil
	}
	
	ctx.Data["operation"] = operation
	ctx.Data["operands"] = []float64{num1, num2}
	ctx.Data["result"] = result
	ctx.Data["input_method"] = "url_path"
	ctx.Data["context_type"] = "noStore_go"
	
	return nil
}

func performCalculation(operation string, num1, num2 float64) (float64, error) {
	switch operation {
	case "add":
		return num1 + num2, nil
	case "subtract":
		return num1 - num2, nil
	case "multiply":
		return num1 * num2, nil
	case "divide":
		if num2 == 0 {
			return 0, fmt.Errorf("Division by zero")
		}
		return num1 / num2, nil
	default:
		return 0, fmt.Errorf("Unsupported operation: %s", operation)
	}
}