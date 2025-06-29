import (
	"fmt"
	"strings"
)

// Run processes file after successful upload
func Run(ctx *EventContext) error {
	// Log file upload
	ctx.Log("File uploaded successfully", map[string]interface{}{
		"id":   ctx.Data["id"],
		"name": ctx.Data["originalName"],
		"size": ctx.Data["size"],
		"user": ctx.Data["uploadedBy"],
	})

	// Emit real-time event for file upload
	if ctx.Emit != nil {
		ctx.Emit("file-uploaded", map[string]interface{}{
			"id":          ctx.Data["id"],
			"name":        ctx.Data["originalName"],
			"size":        ctx.Data["size"],
			"type":        ctx.Data["contentType"],
			"uploadedBy":  ctx.Data["uploadedBy"],
			"uploadedAt":  ctx.Data["uploadedAt"],
		})
	}

	// Add download URL if not already set
	if ctx.Data["url"] == nil {
		ctx.Data["url"] = fmt.Sprintf("/files/%v", ctx.Data["id"])
	}

	// Process images (mark for processing)
	if contentType, ok := ctx.Data["contentType"].(string); ok {
		if strings.HasPrefix(contentType, "image/") {
			// Mark image for thumbnail generation
			ctx.Data["needsThumbnail"] = true
			ctx.Log(fmt.Sprintf("Image file detected, marked for processing: %v", ctx.Data["originalName"]))
		}
	}

	return nil
}