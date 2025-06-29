# WebSocket & Real-time

Go-Deployd provides built-in WebSocket support for real-time applications. Automatically broadcasts collection changes, supports custom events, and scales across multiple server instances.

## Table of Contents

- [Real-time Features](#real-time-features)
- [WebSocket Connection](#websocket-connection)
- [Collection Change Events](#collection-change-events)
- [Custom Events with emit()](#custom-events-with-emit)
- [AfterCommit Events & Response Modification](#aftercommit-events--response-modification)
- [Multi-Server Scaling](#multi-server-scaling)

## Real-time Features

- ✅ Automatic collection change broadcasting
- ✅ Custom event emission from server events
- ✅ Multi-server WebSocket scaling with Redis
- ✅ Connection pooling and load balancing
- ✅ Automatic reconnection and error handling
- ✅ Synchronous AfterCommit events with response modification

## WebSocket Connection

```javascript
// Browser WebSocket connection
const ws = new WebSocket('ws://localhost:2403/ws');

ws.onopen = () => {
    console.log('Connected to go-deployd WebSocket');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Real-time event:', data);
    
    // Handle different event types
    switch(data.type) {
        case 'collection_change':
            handleCollectionChange(data);
            break;
        case 'custom_event':
            handleCustomEvent(data);
            break;
    }
};

function handleCollectionChange(data) {
    console.log(`Collection: ${data.collection}, Action: ${data.action}`);
    console.log('Document:', data.document);
    // Update UI accordingly
}
```

## Collection Change Events

Collection changes (create, update, delete) are automatically broadcast to all connected WebSocket clients. No additional configuration required.

### Event Structure

```json
{
  "type": "collection_change",
  "collection": "todos",
  "action": "created",
  "document": {
    "id": "doc123",
    "title": "New Todo Item",
    "completed": false,
    "createdAt": "2024-06-28T10:00:00Z"
  },
  "timestamp": "2024-06-28T10:00:00Z",
  "userId": "user123"
}
```

### Live Collection Monitoring

```javascript
// Monitor all collection changes
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    if (data.type === 'collection_change') {
        const { collection, action, document } = data;
        
        // Update UI based on collection and action
        switch (collection) {
            case 'todos':
                updateTodoList(action, document);
                break;
            case 'users':
                updateUserList(action, document);
                break;
        }
    }
};

function updateTodoList(action, todo) {
    switch (action) {
        case 'created':
            addTodoToUI(todo);
            break;
        case 'updated':
            updateTodoInUI(todo);
            break;
        case 'deleted':
            removeTodoFromUI(todo.id);
            break;
    }
}
```

### Testing Real-time Updates

Create a document to see real-time updates:

```bash
curl -X POST "http://localhost:2403/todos" \
  -H "Content-Type: application/json" \
  -H "X-Master-Key: your-master-key" \
  -d '{
    "title": "Real-time test todo",
    "completed": false,
    "priority": 1
  }'
```

💡 Open browser console and connect to WebSocket to see this event broadcast live!

## Custom Events with emit()

Use the `emit()` function in server events to send custom real-time notifications to connected clients. Perfect for business logic notifications, progress updates, and alerts.

### Server-Side Event Emission

#### JavaScript Events

```javascript
// aftercommit.js - Emit custom events after successful operations
if (this.priority >= 8) {
    // Emit urgent task alert
    emit('urgent_task_created', {
        taskId: this.id,
        title: this.title,
        priority: this.priority,
        createdBy: me.username,
        timestamp: new Date()
    });
}

if (this.status === 'completed') {
    // Emit completion celebration
    emit('task_completed', {
        taskId: this.id,
        title: this.title,
        completedBy: me.username,
        completionTime: new Date()
    });
}
```

#### Go Events

```go
// aftercommit.go - Emit custom events from Go
package main

type EventHandler struct{}

func (h *EventHandler) Run(ctx interface{}) error {
    eventCtx := ctx.(*EventContext)
    
    // Check if this is a high-priority task
    if priority, ok := eventCtx.Data["priority"].(float64); ok && priority >= 8 {
        // Emit urgent task alert
        eventCtx.Emit("urgent_task_created", map[string]interface{}{
            "taskId":    eventCtx.Data["id"],
            "title":     eventCtx.Data["title"],
            "priority":  priority,
            "createdBy": eventCtx.Me["username"],
            "timestamp": time.Now(),
        })
    }
    
    // Check if task was completed
    if status, ok := eventCtx.Data["status"].(string); ok && status == "completed" {
        eventCtx.Emit("task_completed", map[string]interface{}{
            "taskId":        eventCtx.Data["id"],
            "title":         eventCtx.Data["title"],
            "completedBy":   eventCtx.Me["username"],
            "completionTime": time.Now(),
        })
    }
    
    return nil
}

var EventHandler = &EventHandler{}
```

### Client-Side Custom Event Handling

```javascript
// Listen for custom events from server
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    if (data.type === 'custom_event') {
        switch (data.event) {
            case 'urgent_task_created':
                showUrgentAlert(data.data);
                break;
            case 'task_completed':
                showCompletionCelebration(data.data);
                break;
            case 'user_online':
                updateUserStatus(data.data.userId, 'online');
                break;
        }
    }
};

function showUrgentAlert(taskData) {
    // Show urgent notification
    const notification = new Notification('Urgent Task Created!', {
        body: `${taskData.title} (Priority: ${taskData.priority})`,
        icon: '/urgent-icon.png'
    });
}

function showCompletionCelebration(taskData) {
    // Show completion animation
    console.log(`🎉 ${taskData.completedBy} completed: ${taskData.title}`);
    // Trigger confetti animation, etc.
}
```

### Custom Event Structure

```json
{
  "type": "custom_event",
  "event": "urgent_task_created",
  "data": {
    "taskId": "doc123",
    "title": "Critical System Alert",
    "priority": 9,
    "createdBy": "admin",
    "timestamp": "2024-06-28T10:00:00Z"
  },
  "timestamp": "2024-06-28T10:00:00Z"
}
```

## AfterCommit Events & Response Modification

AfterCommit events run synchronously and can modify the HTTP response before it's sent to the client. This allows for real-time updates while also customizing the API response.

### Response Modification Example

```javascript
// aftercommit.js - Runs AFTER database commit but BEFORE HTTP response
if (this.status === 'completed') {
    // Emit real-time event
    emit('task_completed', {
        taskId: this.id,
        title: this.title,
        completedBy: me.username
    });
    
    // Modify the HTTP response to include additional data
    setResponseData({
        ...this,
        message: 'Congratulations! Task completed successfully!',
        points: calculateCompletionPoints(this.priority),
        nextSuggestion: getNextTaskSuggestion(me.id)
    });
}

// Add real-time notification for high-priority tasks
if (this.priority >= 8) {
    emit('high_priority_task', {
        taskId: this.id,
        title: this.title,
        assignedTo: this.assignedTo
    });
    
    // Add alert to response
    addResponseMessage('High priority task created - notifications sent!');
}
```

### Available Response Functions

- `setResponseData(object)` - Replace the entire response data
- `addResponseField(key, value)` - Add a field to the response
- `addResponseMessage(message)` - Add a message to the response
- `setResponseStatus(code)` - Change the HTTP status code
- `emit(event, data)` - Send real-time event to WebSocket clients

### Real-time + Response Example

Test AfterCommit event with response modification:

```bash
curl -X POST "http://localhost:2403/todos" \
  -H "Content-Type: application/json" \
  -H "X-Master-Key: your-master-key" \
  -d '{
    "title": "Critical system maintenance",
    "priority": 9,
    "assignedTo": "admin"
  }'
```

💡 Both WebSocket clients and the HTTP response will receive enhanced data!

## Multi-Server Scaling

For production deployments with multiple server instances:

```bash
# Set Redis URL for multi-server WebSocket scaling
export REDIS_URL="redis://localhost:6379"

# Start multiple server instances
./deployd --port 2403 &
./deployd --port 2404 &
./deployd --port 2405 &

# WebSocket events will be synchronized across all instances
```

With Redis configured, WebSocket events are automatically synchronized across all server instances, enabling horizontal scaling while maintaining real-time functionality.