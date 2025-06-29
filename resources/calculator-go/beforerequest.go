// BeforeRequest for calculator-go
func Run(ctx *EventContext) error {
	ctx.Log("Calculator Go BeforeRequest called")
	
	// This should be available for all collections
	return nil
}