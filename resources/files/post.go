import (
	"fmt"
)

// Run validates file uploads before they are processed
func Run(ctx *EventContext) error {
	// Example: Reject files larger than 10MB (uncomment to enable)
	// if size, ok := ctx.Data["size"].(float64); ok && size > 10*1024*1024 {
	// 	ctx.Cancel("File size exceeds 10MB limit", 400)
	// 	return nil
	// }

	// Example: Only allow certain file types (uncomment to enable)
	// allowedTypes := []string{
	// 	"image/jpeg",
	// 	"image/png",
	// 	"image/gif",
	// 	"application/pdf",
	// 	"application/msword",
	// 	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	// }
	// 
	// if contentType, ok := ctx.Data["contentType"].(string); ok {
	// 	allowed := false
	// 	for _, t := range allowedTypes {
	// 		if t == contentType {
	// 			allowed = true
	// 			break
	// 		}
	// 	}
	// 	if !allowed {
	// 		ctx.Cancel(fmt.Sprintf("File type not allowed. Allowed types: %s", strings.Join(allowedTypes, ", ")), 400)
	// 		return nil
	// 	}
	// }

	// Require authentication for uploads
	if ctx.Me == nil || ctx.Me["id"] == nil {
		ctx.Cancel("Authentication required for file uploads", 401)
		return nil
	}

	// Add custom metadata
	ctx.Data["uploadedAt"] = "now" // Will be set by storage manager
	ctx.Data["uploadedBy"] = ctx.Me["id"]

	// Sanitize filename
	if originalName, ok := ctx.Data["originalName"].(string); ok {
		// Replace any non-alphanumeric characters (except dots and hyphens) with underscores
		sanitized := ""
		for _, r := range originalName {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
				sanitized += string(r)
			} else {
				sanitized += "_"
			}
		}
		ctx.Data["originalName"] = sanitized
	}

	ctx.Log(fmt.Sprintf("File upload validated: %v", ctx.Data["originalName"]))
	return nil
}