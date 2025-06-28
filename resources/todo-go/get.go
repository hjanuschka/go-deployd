import "time"

// Run processes todo data when retrieved
func Run(ctx *EventContext) error {
	// Add computed status field
	completed, _ := ctx.Data["completed"].(bool)
	if completed {
		ctx.Data["status"] = "Done"
	} else {
		ctx.Data["status"] = "Pending"
	}

	// Format dates for better display
	if createdAt, ok := ctx.Data["createdAt"].(time.Time); ok {
		ctx.Data["formattedDate"] = createdAt.Format("2006-01-02 15:04")
	}

	// Add priority label
	if priority, ok := ctx.Data["priority"].(float64); ok {
		switch int(priority) {
		case 1:
			ctx.Data["priorityLabel"] = "Low"
		case 2:
			ctx.Data["priorityLabel"] = "Normal"
		case 3:
			ctx.Data["priorityLabel"] = "High"
		case 4:
			ctx.Data["priorityLabel"] = "Urgent"
		case 5:
			ctx.Data["priorityLabel"] = "Critical"
		default:
			ctx.Data["priorityLabel"] = "Normal"
		}
	}

	return nil
}
