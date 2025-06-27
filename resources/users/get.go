//go:build ignore
// +build ignore

package main

// Run filters or modifies retrieved documents
func Run(ctx *EventContext) error {
	// Hide sensitive fields (syntax sugar for delete)
	ctx.Hide("password")
	ctx.Hide("verificationToken")

	return nil
}
