# Internal API

The internal API allows event scripts to access and modify data from other collections within the same go-deployd instance. This enables powerful cross-collection operations without making external HTTP requests.

## Overview

The internal API is available through the `ctx.Dpd` object in Go events and the `dpd` object in JavaScript events. It provides full HTTP-like semantics, executing the complete request pipeline including all events and validations.

## Available Methods

### Collection Access
```go
// Go
client := ctx.Dpd.Collection("collectionName")

// JavaScript
const client = dpd.collectionName;
```

### GET - Retrieve Documents
```go
// Get a single document by ID
user, err := ctx.Dpd.Collection("users").Get("userId123", nil)

// Get with query parameters
posts, err := ctx.Dpd.Collection("posts").Get("", map[string]interface{}{
    "$limit": 10,
    "$sort": map[string]interface{}{"createdAt": -1},
})
```

```javascript
// JavaScript
// Get a single document
const user = await dpd.users.get("userId123");

// Get with query
const posts = await dpd.posts.get({
    $limit: 10,
    $sort: {createdAt: -1}
});
```

### POST - Create Documents
```go
// Go
newPost, err := ctx.Dpd.Collection("posts").Post(map[string]interface{}{
    "title": "New Post",
    "content": "Post content",
    "userId": ctx.Me["id"],
})
```

```javascript
// JavaScript
const newPost = await dpd.posts.post({
    title: "New Post",
    content: "Post content",
    userId: context.me.id
});
```

### PUT - Update Documents
```go
// Go
updated, err := ctx.Dpd.Collection("posts").Put("postId123", map[string]interface{}{
    "title": "Updated Title",
    "updatedAt": time.Now(),
})
```

```javascript
// JavaScript
const updated = await dpd.posts.put("postId123", {
    title: "Updated Title",
    updatedAt: Date.now()
});
```

### DELETE - Remove Documents
```go
// Go
err := ctx.Dpd.Collection("posts").Delete("postId123")
```

```javascript
// JavaScript
await dpd.posts.del("postId123");
```

## Error Handling

All internal API methods return errors when operations fail:

```go
// Go
user, err := ctx.Dpd.Collection("users").Get(userId, nil)
if err != nil {
    ctx.Log("Failed to fetch user", map[string]interface{}{
        "error": err.Error(),
    })
    // Handle error appropriately
    return nil
}
```

```javascript
// JavaScript
try {
    const user = await dpd.users.get(userId);
} catch (err) {
    context.log("Failed to fetch user", {error: err.message});
    // Handle error
}
```

## Important Notes

1. **Event Execution**: Internal API calls execute the full event pipeline, including validation, beforeRequest, afterRequest, etc.
2. **Permissions**: Internal API calls bypass authentication but respect other validation rules
3. **Circular Dependencies**: Be careful to avoid infinite loops when collections reference each other
4. **Performance**: Internal API calls are faster than HTTP requests but still execute the full pipeline

## Common Use Cases

### 1. Data Enrichment
Enrich documents with related data from other collections:

```go
// In posts/get.go
func Run(ctx *EventContext) error {
    userId := ctx.Data["userId"].(string)
    
    // Fetch author details
    user, err := ctx.Dpd.Collection("users").Get(userId, nil)
    if err == nil && user != nil {
        if userData, ok := user.(map[string]interface{}); ok {
            ctx.Data["author"] = map[string]interface{}{
                "id":       userData["id"],
                "username": userData["username"],
                "email":    userData["email"],
            }
        }
    }
    
    return nil
}
```

### 2. Cross-Collection Validation
Validate references between collections:

```go
// In posts/validate.go
func Run(ctx *EventContext) error {
    userId := ctx.Data["userId"].(string)
    
    // Verify user exists and is active
    user, err := ctx.Dpd.Collection("users").Get(userId, nil)
    if err != nil {
        ctx.Error("userId", "User not found")
        return nil
    }
    
    if userData, ok := user.(map[string]interface{}); ok {
        if active, ok := userData["active"].(bool); ok && !active {
            ctx.Error("userId", "User account is not active")
        }
    }
    
    return nil
}
```

### 3. Cascading Operations
Update related documents when changes occur:

```go
// In users/afterdelete.go
func Run(ctx *EventContext) error {
    userId := ctx.Data["id"].(string)
    
    // Delete all posts by this user
    posts, err := ctx.Dpd.Collection("posts").Get("", map[string]interface{}{
        "userId": userId,
    })
    
    if err == nil && posts != nil {
        if postList, ok := posts.([]interface{}); ok {
            for _, post := range postList {
                if postData, ok := post.(map[string]interface{}); ok {
                    postId := postData["id"].(string)
                    ctx.Dpd.Collection("posts").Delete(postId)
                }
            }
        }
    }
    
    return nil
}
```

### 4. Aggregations and Statistics
Calculate statistics across collections:

```go
// In stats/get.go (noStore collection)
func Run(ctx *EventContext) error {
    // Count total users
    users, _ := ctx.Dpd.Collection("users").Get("", map[string]interface{}{
        "$limit": 1000,
    })
    
    userCount := 0
    if userList, ok := users.([]interface{}); ok {
        userCount = len(userList)
    }
    
    // Count total posts
    posts, _ := ctx.Dpd.Collection("posts").Get("", map[string]interface{}{
        "$limit": 1000,
    })
    
    postCount := 0
    if postList, ok := posts.([]interface{}); ok {
        postCount = len(postList)
    }
    
    ctx.Data["stats"] = map[string]interface{}{
        "totalUsers": userCount,
        "totalPosts": postCount,
    }
    
    return nil
}
```

## JavaScript Examples

```javascript
// In posts/get.js
function Run(context) {
    // Enrich with author data
    if (context.data.userId) {
        dpd.users.get(context.data.userId)
            .then(user => {
                context.data.author = {
                    id: user.id,
                    username: user.username,
                    email: user.email
                };
            })
            .catch(err => {
                context.log("Failed to fetch author", {error: err.message});
            });
    }
}
```

## Best Practices

1. **Error Handling**: Always handle errors from internal API calls
2. **Data Validation**: Validate data existence before using it
3. **Logging**: Log operations for debugging, especially in development
4. **Performance**: Cache results when making multiple calls to the same collection
5. **Atomicity**: Remember that operations are not transactional across collections