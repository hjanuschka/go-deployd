import (
	"fmt"
)

// Run validates file deletion
func Run(ctx *EventContext) error {
	// Only allow file owner or admin to delete (keep this enabled for security)
	if !ctx.IsRoot && ctx.Data["uploadedBy"] != ctx.Me["id"] {
		ctx.Cancel("You can only delete your own files", 403)
		return nil
	}

	// Example: Prevent deletion of files uploaded in the last hour (uncomment to enable)
	// if uploadedAt, ok := ctx.Data["uploadedAt"].(string); ok {
	// 	uploadTime, err := time.Parse(time.RFC3339, uploadedAt)
	// 	if err == nil {
	// 		hourAgo := time.Now().Add(-time.Hour)
	// 		if uploadTime.After(hourAgo) && !ctx.IsRoot {
	// 			ctx.Cancel("Cannot delete files uploaded within the last hour", 400)
	// 			return nil
	// 		}
	// 	}
	// }

	deletedBy := ""
	if ctx.Me != nil && ctx.Me["id"] != nil {
		deletedBy = fmt.Sprintf("%v", ctx.Me["id"])
	}

	ctx.Log("File deletion authorized", map[string]interface{}{
		"id":        ctx.Data["id"],
		"name":      ctx.Data["originalName"],
		"deletedBy": deletedBy,
	})

	return nil
}