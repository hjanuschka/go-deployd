import (
	"fmt"
	"time"
)

// Run processes the AfterCommit event - called after document is saved to database
// Can modify the response document before it's sent to the client
func Run(ctx *EventContext) error {
	// Log the AfterCommit event
	ctx.Log("AfterCommit event started for todo-go", map[string]interface{}{
		"event": "aftercommit",
		"action": "response_modification",
	})
	
	// Add fields to demonstrate response modification
	ctx.Data["afterCommitProcessed"] = true
	ctx.Data["afterCommitTimestamp"] = time.Now().Format(time.RFC3339)
	
	// Add a custom field based on document data
	if title, ok := ctx.Data["title"].(string); ok {
		ctx.Data["processedTitle"] = fmt.Sprintf("✅ Processed: %s", title)
	}
	
	// Get the document ID and add a custom message
	if docID, ok := ctx.Data["id"].(string); ok {
		ctx.Data["customMessage"] = fmt.Sprintf("Document %s processed by AfterCommit", docID)
	}
	
	// Add a calculated field
	if priority, ok := ctx.Data["priority"].(float64); ok {
		ctx.Data["priorityLabel"] = getPriorityLabel(int(priority))
	}
	
	ctx.Log("AfterCommit event completed - response modified", map[string]interface{}{
		"fieldsAdded": []string{"afterCommitProcessed", "afterCommitTimestamp", "processedTitle", "customMessage", "priorityLabel"},
	})
	
	return nil
}

// Helper function to convert priority number to label
func getPriorityLabel(priority int) string {
	switch priority {
	case 1:
		return "Low"
	case 2, 3:
		return "Medium"
	case 4, 5:
		return "High"
	default:
		return "Critical"
	}
}