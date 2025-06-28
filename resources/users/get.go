//go:build ignore
// +build ignore

package main

// EventContext stub for compilation
type EventContext struct {
	Data map[string]interface{}
}

func (ctx *EventContext) Hide(field string) {}
func (ctx *EventContext) Cancel(message string, code int) {}

// Run filters or modifies retrieved documents
func Run(ctx *EventContext) error {
	// Hide sensitive fields (syntax sugar for delete)
	ctx.Hide("password")
	ctx.Hide("verificationToken")

	return nil
}
