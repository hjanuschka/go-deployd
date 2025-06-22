import React, { useState } from 'react'
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalCloseButton,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  VStack,
  HStack,
  Text,
  Heading,
  Code,
  Card,
  CardBody,
  Box,
  Badge,
  Alert,
  AlertIcon,
  AlertDescription,
  Divider,
  useColorModeValue,
} from '@chakra-ui/react'
import CodeMirror from '@uiw/react-codemirror'
import { javascript } from '@codemirror/lang-javascript'
import { go } from '@codemirror/lang-go'
import { oneDark } from '@codemirror/theme-one-dark'

const EventDocumentation = ({ isOpen, onClose }) => {
  const [selectedLang, setSelectedLang] = useState('js')
  const cardBg = useColorModeValue('gray.50', 'gray.700')

  const jsExamples = {
    get: `// GET Event - Filter and modify retrieved documents
// Available: this (current document), me (current user), query, cancel(), hide()

// Example 1: Hide sensitive data from non-admin users
if (!isRoot) {
  hide('email');
  hide('internalNotes');
  hide('cost');
}

// Example 2: Only show user's own documents
if (!isRoot && me) {
  if (this.userId !== me.id) {
    cancel("Not authorized to view this document", 403);
  }
}

// Example 3: Add computed fields
this.displayName = this.firstName + ' ' + this.lastName;
this.isOwner = me && this.userId === me.id;`,

    validate: `// VALIDATE Event - Validate data before saving (POST/PUT)
// Available: this (document data), me, error(), cancel(), hasErrors()

// Example 1: Required field validation
if (!this.title || this.title.trim() === '') {
  error('title', 'Title is required');
}

if (!this.email) {
  error('email', 'Email is required');
} else if (!/^[^@]+@[^@]+\\.[^@]+$/.test(this.email)) {
  error('email', 'Invalid email format');
}

// Example 2: Custom business logic validation
if (this.priority && (this.priority < 1 || this.priority > 5)) {
  error('priority', 'Priority must be between 1 and 5');
}

if (this.startDate && this.endDate) {
  if (new Date(this.startDate) >= new Date(this.endDate)) {
    error('endDate', 'End date must be after start date');
  }
}

// Example 3: Conditional validation
if (this.type === 'premium' && !this.subscriptionId) {
  error('subscriptionId', 'Subscription ID required for premium accounts');
}`,

    post: `// POST Event - Modify data when creating new documents
// Available: this (document data), me, cancel(), error()

// Example 1: Set creation defaults
this.createdAt = new Date().toISOString();
this.completed = this.completed || false;
this.status = this.status || 'active';

// Example 2: Add user information
if (me) {
  this.createdBy = me.id;
  this.createdByName = me.username;
} else {
  cancel("Authentication required", 401);
}

// Example 3: Generate unique identifiers
this.slug = this.title.toLowerCase()
  .replace(/[^a-z0-9]+/g, '-')
  .replace(/^-+|-+$/g, '');

// Add random suffix if needed
this.confirmationCode = Math.random().toString(36).substring(2, 15);

// Example 4: Validate and set relationships
if (this.categoryId) {
  // Note: In real apps, you'd check if category exists
  // dpd.categories.first({id: this.categoryId}, function(err, category) { ... })
}`,

    put: `// PUT Event - Modify data when updating documents
// Available: this (new data), previous (old data), me, cancel(), error()

// Example 1: Track modifications
this.updatedAt = new Date().toISOString();
if (me) {
  this.updatedBy = me.id;
}

// Example 2: Preserve certain fields
if (previous.createdAt) {
  this.createdAt = previous.createdAt;
  this.createdBy = previous.createdBy;
}

// Example 3: Validate ownership for updates
if (!isRoot && me && previous.userId !== me.id) {
  cancel("You can only update your own documents", 403);
}

// Example 4: Conditional updates based on status
if (previous.status === 'published' && this.status !== 'published') {
  this.unpublishedAt = new Date().toISOString();
  this.unpublishedBy = me.id;
}

// Example 5: Prevent certain changes
if (previous.type === 'system' && !isRoot) {
  cancel("System documents can only be modified by administrators", 403);
}`,

    delete: `// DELETE Event - Control document deletion
// Available: this (document to delete), me, cancel(), error()

// Example 1: Prevent deletion of certain documents
if (this.protected === true) {
  cancel("This document is protected and cannot be deleted", 403);
}

// Example 2: Ownership validation
if (!isRoot && me && this.userId !== me.id) {
  cancel("You can only delete your own documents", 403);
}

// Example 3: Cascade delete related documents
// Note: Actual deletion happens in aftercommit event
if (this.type === 'project') {
  // Mark related tasks for deletion
  this._deleteRelatedTasks = true;
}

// Example 4: Archive instead of delete
if (this.type === 'important') {
  // Don't actually delete, just archive
  dpd.archive.post({
    originalId: this.id,
    originalCollection: 'todos',
    data: this,
    archivedAt: new Date().toISOString(),
    archivedBy: me ? me.id : null
  });
  
  // Cancel the actual deletion
  cancel("Document archived instead of deleted", 200);
}`,

    aftercommit: `// AFTERCOMMIT Event - Runs after database changes are committed
// Available: this (document), me, query, result

// Example 1: Send notifications
if (query.method === 'POST') {
  // New document created - send notification
  dpd.notifications.post({
    type: 'new_document',
    documentId: this.id,
    userId: this.createdBy,
    message: 'New document: ' + this.title
  });
}

// Example 2: Update related counters
if (query.method === 'POST' && this.categoryId) {
  dpd.categories.update(this.categoryId, {
    $inc: { documentCount: 1 }
  });
}

if (query.method === 'DELETE' && this.categoryId) {
  dpd.categories.update(this.categoryId, {
    $inc: { documentCount: -1 }
  });
}

// Example 3: Log activity
dpd.activitylog.post({
  action: query.method,
  collection: 'todos',
  documentId: this.id,
  userId: me ? me.id : null,
  timestamp: new Date().toISOString(),
  details: {
    title: this.title,
    changes: query.method === 'PUT' ? result.changes : null
  }
});

// Example 4: Clear caches or trigger background jobs
if (this.status === 'published') {
  // Trigger cache invalidation
  dpd.cache.delete({ key: 'published_documents' });
  
  // Queue background job
  dpd.jobs.post({
    type: 'index_document',
    documentId: this.id,
    priority: 'high'
  });
}`,

    beforerequest: `// BEFOREREQUEST Event - Runs before any HTTP request
// Available: query, url, me, cancel(), error()

// Example 1: API rate limiting
if (!isRoot && me) {
  var requestCount = me.requestCount || 0;
  var lastRequest = me.lastRequestTime || 0;
  var now = new Date().getTime();
  
  // Reset counter if more than 1 hour passed
  if (now - lastRequest > 3600000) {
    requestCount = 0;
  }
  
  // Check rate limit (100 requests per hour)
  if (requestCount >= 100) {
    cancel("Rate limit exceeded. Try again later.", 429);
  }
  
  // Update counter
  dpd.users.update(me.id, {
    requestCount: requestCount + 1,
    lastRequestTime: now
  });
}

// Example 2: Maintenance mode
var maintenanceMode = false; // This could come from a config
if (maintenanceMode && !isRoot) {
  cancel("System is under maintenance. Please try again later.", 503);
}

// Example 3: Request logging and analytics
dpd.requestlog.post({
  method: query.method || 'GET',
  url: url,
  userId: me ? me.id : null,
  userAgent: query.userAgent,
  ip: query.ip,
  timestamp: new Date().toISOString()
});

// Example 4: Security headers and CORS
if (query.method === 'OPTIONS') {
  // Handle preflight requests
  headers['Access-Control-Allow-Origin'] = '*';
  headers['Access-Control-Allow-Methods'] = 'GET, POST, PUT, DELETE';
  headers['Access-Control-Allow-Headers'] = 'Content-Type, Authorization';
}

// Example 5: API versioning
if (url.startsWith('/api/v1/')) {
  // Legacy API compatibility
  query.apiVersion = 'v1';
} else if (url.startsWith('/api/v2/')) {
  query.apiVersion = 'v2';
}`
  }

  const goExamples = {
    get: `// GET Event - Filter and modify retrieved documents (Go)
package main

import (
	"github.com/hjanuschka/go-deployd/internal/events"
)

func Run(ctx *events.EventContext) error {
	// Example 1: Hide sensitive data from non-admin users
	if !ctx.IsRoot {
		ctx.Hide("email")
		ctx.Hide("internalNotes")
		ctx.Hide("cost")
	}

	// Example 2: Only show user's own documents
	if !ctx.IsRoot && ctx.Me != nil {
		if userId, ok := ctx.Data["userId"].(string); ok {
			if meId, exists := ctx.Me["id"].(string); exists && userId != meId {
				return ctx.Cancel("Not authorized to view this document", 403)
			}
		}
	}

	// Example 3: Add computed fields
	if firstName, ok := ctx.Data["firstName"].(string); ok {
		if lastName, ok := ctx.Data["lastName"].(string); ok {
			ctx.Data["displayName"] = firstName + " " + lastName
		}
	}

	if ctx.Me != nil {
		if userId, ok := ctx.Data["userId"].(string); ok {
			if meId, exists := ctx.Me["id"].(string); exists {
				ctx.Data["isOwner"] = userId == meId
			}
		}
	}

	return nil
}`,

    validate: `// VALIDATE Event - Validate data before saving (Go)
package main

import (
	"regexp"
	"strings"
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

func Run(ctx *events.EventContext) error {
	// Example 1: Required field validation
	if title, ok := ctx.Data["title"].(string); !ok || strings.TrimSpace(title) == "" {
		ctx.Error("title", "Title is required")
	}

	if email, ok := ctx.Data["email"].(string); !ok || email == "" {
		ctx.Error("email", "Email is required")
	} else {
		emailRegex := regexp.MustCompile(` + "`^[^@]+@[^@]+\\.[^@]+$`" + `)
		if !emailRegex.MatchString(email) {
			ctx.Error("email", "Invalid email format")
		}
	}

	// Example 2: Custom business logic validation
	if priority, ok := ctx.Data["priority"].(float64); ok {
		if priority < 1 || priority > 5 {
			ctx.Error("priority", "Priority must be between 1 and 5")
		}
	}

	// Example 3: Date validation
	if startDateStr, ok := ctx.Data["startDate"].(string); ok {
		if endDateStr, ok := ctx.Data["endDate"].(string); ok {
			startDate, err1 := time.Parse(time.RFC3339, startDateStr)
			endDate, err2 := time.Parse(time.RFC3339, endDateStr)
			
			if err1 == nil && err2 == nil && !startDate.Before(endDate) {
				ctx.Error("endDate", "End date must be after start date")
			}
		}
	}

	// Example 4: Conditional validation
	if docType, ok := ctx.Data["type"].(string); ok && docType == "premium" {
		if subscriptionId, exists := ctx.Data["subscriptionId"].(string); !exists || subscriptionId == "" {
			ctx.Error("subscriptionId", "Subscription ID required for premium accounts")
		}
	}

	return nil
}`,

    post: `// POST Event - Modify data when creating new documents (Go)
package main

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

func Run(ctx *events.EventContext) error {
	// Example 1: Set creation defaults
	ctx.Data["createdAt"] = time.Now().Format(time.RFC3339)
	
	if _, exists := ctx.Data["completed"]; !exists {
		ctx.Data["completed"] = false
	}
	
	if _, exists := ctx.Data["status"]; !exists {
		ctx.Data["status"] = "active"
	}

	// Example 2: Add user information
	if ctx.Me != nil {
		if meId, exists := ctx.Me["id"].(string); exists {
			ctx.Data["createdBy"] = meId
		}
		if username, exists := ctx.Me["username"].(string); exists {
			ctx.Data["createdByName"] = username
		}
	} else {
		return ctx.Cancel("Authentication required", 401)
	}

	// Example 3: Generate unique identifiers
	if title, ok := ctx.Data["title"].(string); ok {
		// Create slug from title
		slug := strings.ToLower(title)
		reg := regexp.MustCompile(` + "`[^a-z0-9]+`" + `)
		slug = reg.ReplaceAllString(slug, "-")
		slug = strings.Trim(slug, "-")
		ctx.Data["slug"] = slug
	}

	// Generate random confirmation code
	bytes := make([]byte, 8)
	rand.Read(bytes)
	ctx.Data["confirmationCode"] = hex.EncodeToString(bytes)

	return nil
}`,

    put: `// PUT Event - Modify data when updating documents (Go)
package main

import (
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

func Run(ctx *events.EventContext) error {
	// Example 1: Track modifications
	ctx.Data["updatedAt"] = time.Now().Format(time.RFC3339)
	
	if ctx.Me != nil {
		if meId, exists := ctx.Me["id"].(string); exists {
			ctx.Data["updatedBy"] = meId
		}
	}

	// Example 2: Preserve certain fields
	if ctx.Previous != nil {
		if createdAt, exists := ctx.Previous["createdAt"]; exists {
			ctx.Data["createdAt"] = createdAt
		}
		if createdBy, exists := ctx.Previous["createdBy"]; exists {
			ctx.Data["createdBy"] = createdBy
		}
	}

	// Example 3: Validate ownership for updates
	if !ctx.IsRoot && ctx.Me != nil && ctx.Previous != nil {
		if prevUserId, ok := ctx.Previous["userId"].(string); ok {
			if meId, exists := ctx.Me["id"].(string); exists && prevUserId != meId {
				return ctx.Cancel("You can only update your own documents", 403)
			}
		}
	}

	// Example 4: Conditional updates based on status
	if ctx.Previous != nil {
		if prevStatus, ok := ctx.Previous["status"].(string); ok && prevStatus == "published" {
			if newStatus, ok := ctx.Data["status"].(string); ok && newStatus != "published" {
				ctx.Data["unpublishedAt"] = time.Now().Format(time.RFC3339)
				if ctx.Me != nil {
					if meId, exists := ctx.Me["id"].(string); exists {
						ctx.Data["unpublishedBy"] = meId
					}
				}
			}
		}
	}

	// Example 5: Prevent certain changes
	if ctx.Previous != nil {
		if prevType, ok := ctx.Previous["type"].(string); ok && prevType == "system" && !ctx.IsRoot {
			return ctx.Cancel("System documents can only be modified by administrators", 403)
		}
	}

	return nil
}`,

    delete: `// DELETE Event - Control document deletion (Go)
package main

import (
	"github.com/hjanuschka/go-deployd/internal/events"
)

func Run(ctx *events.EventContext) error {
	// Example 1: Prevent deletion of certain documents
	if protected, ok := ctx.Data["protected"].(bool); ok && protected {
		return ctx.Cancel("This document is protected and cannot be deleted", 403)
	}

	// Example 2: Ownership validation
	if !ctx.IsRoot && ctx.Me != nil {
		if userId, ok := ctx.Data["userId"].(string); ok {
			if meId, exists := ctx.Me["id"].(string); exists && userId != meId {
				return ctx.Cancel("You can only delete your own documents", 403)
			}
		}
	}

	// Example 3: Mark related documents for cascade deletion
	if docType, ok := ctx.Data["type"].(string); ok && docType == "project" {
		// Mark for cleanup in aftercommit event
		ctx.Data["_deleteRelatedTasks"] = true
	}

	// Example 4: Archive instead of delete
	if docType, ok := ctx.Data["type"].(string); ok && docType == "important" {
		// Create archive record (this would need actual implementation)
		// archiveData := map[string]interface{}{
		//     "originalId": ctx.Data["id"],
		//     "originalCollection": "todos",
		//     "data": ctx.Data,
		//     "archivedAt": time.Now().Format(time.RFC3339),
		// }
		
		// Don't actually delete, just archive
		return ctx.Cancel("Document archived instead of deleted", 200)
	}

	return nil
}`,

    aftercommit: `// AFTERCOMMIT Event - Runs after database changes (Go)
package main

import (
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

func Run(ctx *events.EventContext) error {
	// Example 1: Send notifications based on operation
	switch ctx.Method {
	case "POST":
		// New document created - could send notification
		// (In real implementation, you'd use the internal client)
		// notificationData := map[string]interface{}{
		//     "type": "new_document",
		//     "documentId": ctx.Data["id"],
		//     "userId": ctx.Data["createdBy"],
		//     "message": "New document: " + ctx.Data["title"].(string),
		// }
		
	case "PUT":
		// Document updated
		// Similar notification logic
		
	case "DELETE":
		// Document deleted
		// Cleanup or notification logic
	}

	// Example 2: Update related counters
	if categoryId, ok := ctx.Data["categoryId"].(string); ok && categoryId != "" {
		switch ctx.Method {
		case "POST":
			// Increment category document count
			// dpd.categories.update(categoryId, {$inc: {documentCount: 1}})
			
		case "DELETE":
			// Decrement category document count
			// dpd.categories.update(categoryId, {$inc: {documentCount: -1}})
		}
	}

	// Example 3: Log activity
	// activityData := map[string]interface{}{
	//     "action": ctx.Method,
	//     "collection": "todos",
	//     "documentId": ctx.Data["id"],
	//     "userId": getUserId(ctx),
	//     "timestamp": time.Now().Format(time.RFC3339),
	//     "details": map[string]interface{}{
	//         "title": ctx.Data["title"],
	//     },
	// }

	// Example 4: Clear caches or trigger background jobs
	if status, ok := ctx.Data["status"].(string); ok && status == "published" {
		// Clear cache
		// dpd.cache.delete({key: "published_documents"})
		
		// Queue background job
		// jobData := map[string]interface{}{
		//     "type": "index_document",
		//     "documentId": ctx.Data["id"],
		//     "priority": "high",
		// }
	}

	return nil
}

func getUserId(ctx *events.EventContext) interface{} {
	if ctx.Me != nil {
		if userId, exists := ctx.Me["id"]; exists {
			return userId
		}
	}
	return nil
}`,

    beforerequest: `// BEFOREREQUEST Event - Runs before any HTTP request (Go)
package main

import (
	"strings"
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

func Run(ctx *events.EventContext) error {
	// Example 1: API rate limiting
	if !ctx.IsRoot && ctx.Me != nil {
		var requestCount float64
		var lastRequestTime int64
		
		if count, ok := ctx.Me["requestCount"].(float64); ok {
			requestCount = count
		}
		
		if lastTime, ok := ctx.Me["lastRequestTime"].(int64); ok {
			lastRequestTime = lastTime
		}
		
		now := time.Now().Unix()
		
		// Reset counter if more than 1 hour passed
		if now-lastRequestTime > 3600 {
			requestCount = 0
		}
		
		// Check rate limit (100 requests per hour)
		if requestCount >= 100 {
			return ctx.Cancel("Rate limit exceeded. Try again later.", 429)
		}
		
		// Note: In real implementation, you'd update user record
		// dpd.users.update(ctx.Me["id"], {
		//     requestCount: requestCount + 1,
		//     lastRequestTime: now,
		// })
	}

	// Example 2: Maintenance mode
	maintenanceMode := false // This could come from a config collection
	if maintenanceMode && !ctx.IsRoot {
		return ctx.Cancel("System is under maintenance. Please try again later.", 503)
	}

	// Example 3: Request logging and analytics
	// logData := map[string]interface{}{
	//     "method": ctx.Method,
	//     "url": ctx.URL,
	//     "userId": getUserId(ctx),
	//     "userAgent": ctx.Query["userAgent"],
	//     "ip": ctx.Query["ip"],
	//     "timestamp": time.Now().Format(time.RFC3339),
	// }
	// dpd.requestlog.post(logData)

	// Example 4: API versioning
	if strings.HasPrefix(ctx.URL, "/api/v1/") {
		// Legacy API compatibility
		ctx.Query["apiVersion"] = "v1"
	} else if strings.HasPrefix(ctx.URL, "/api/v2/") {
		ctx.Query["apiVersion"] = "v2"
	}

	// Example 5: Security and CORS
	if ctx.Method == "OPTIONS" {
		// Handle preflight requests
		// headers["Access-Control-Allow-Origin"] = "*"
		// headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, DELETE"
		// headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization"
	}

	return nil
}`
  }

  const apiReference = {
    js: {
      objects: [
        { name: 'this', desc: 'The current document being processed' },
        { name: 'me', desc: 'The current authenticated user (null if not logged in)' },
        { name: 'query', desc: 'Query parameters and request data' },
        { name: 'previous', desc: 'Previous document state (PUT/DELETE events only)' },
        { name: 'isRoot', desc: 'Boolean indicating if current user has admin privileges' }
      ],
      functions: [
        { name: 'error(field, message)', desc: 'Add validation error for a field' },
        { name: 'cancel(message, statusCode)', desc: 'Cancel the request with error message' },
        { name: 'hide(field)', desc: 'Hide field from response (GET events only)' },
        { name: 'protect(field)', desc: 'Prevent field from being modified' },
        { name: 'hasErrors()', desc: 'Check if any validation errors exist' }
      ]
    },
    go: {
      objects: [
        { name: 'ctx.Data', desc: 'Map containing the current document data' },
        { name: 'ctx.Me', desc: 'Map containing current authenticated user data' },
        { name: 'ctx.Query', desc: 'Map containing query parameters and request data' },
        { name: 'ctx.Previous', desc: 'Map containing previous document state (PUT/DELETE events)' },
        { name: 'ctx.IsRoot', desc: 'Boolean indicating if current user has admin privileges' }
      ],
      functions: [
        { name: 'ctx.Error(field, message)', desc: 'Add validation error for a field' },
        { name: 'ctx.Cancel(message, statusCode)', desc: 'Cancel the request and return error' },
        { name: 'ctx.Hide(field)', desc: 'Hide field from response (GET events only)' },
        { name: 'ctx.Protect(field)', desc: 'Prevent field from being modified' }
      ]
    }
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="6xl">
      <ModalOverlay />
      <ModalContent maxH="90vh">
        <ModalHeader>
          <HStack>
            <Text>Event Scripts Documentation</Text>
            <Badge colorScheme="blue">Complete Reference</Badge>
          </HStack>
        </ModalHeader>
        <ModalCloseButton />
        <ModalBody overflowY="auto" pb={6}>
          <Tabs>
            <TabList>
              <Tab>Event Types</Tab>
              <Tab>JavaScript Examples</Tab>
              <Tab>Go Examples</Tab>
              <Tab>API Reference</Tab>
            </TabList>

            <TabPanels>
              {/* Event Types Tab */}
              <TabPanel>
                <VStack align="stretch" spacing={4}>
                  <Alert status="info">
                    <AlertIcon />
                    <AlertDescription>
                      Events are functions that run at specific points in the request lifecycle. 
                      Choose JavaScript for quick scripting or Go for better performance.
                    </AlertDescription>
                  </Alert>

                  <VStack align="stretch" spacing={3}>
                    {[
                      { name: 'beforerequest', title: 'Before Request', desc: 'Runs before any HTTP request. Use for authentication, rate limiting, and request logging.' },
                      { name: 'get', title: 'On GET', desc: 'Runs when retrieving documents. Use to filter results, hide sensitive data, or add computed fields.' },
                      { name: 'validate', title: 'On Validate', desc: 'Runs before saving (POST/PUT). Use for data validation and business rule enforcement.' },
                      { name: 'post', title: 'On POST', desc: 'Runs when creating new documents. Use to set defaults, add metadata, or enforce creation rules.' },
                      { name: 'put', title: 'On PUT', desc: 'Runs when updating documents. Use to track changes, validate updates, or preserve certain fields.' },
                      { name: 'delete', title: 'On DELETE', desc: 'Runs before deleting documents. Use to prevent deletion or implement soft deletes.' },
                      { name: 'aftercommit', title: 'After Commit', desc: 'Runs after database changes are committed. Use for notifications, cache updates, or logging.' }
                    ].map(event => (
                      <Card key={event.name} bg={cardBg}>
                        <CardBody>
                          <HStack align="start" spacing={4}>
                            <Badge colorScheme="brand" fontSize="sm" px={3} py={1}>
                              {event.name}
                            </Badge>
                            <Box flex="1">
                              <Heading size="sm" mb={1}>{event.title}</Heading>
                              <Text fontSize="sm" color="gray.600">{event.desc}</Text>
                            </Box>
                          </HStack>
                        </CardBody>
                      </Card>
                    ))}
                  </VStack>
                </VStack>
              </TabPanel>

              {/* JavaScript Examples Tab */}
              <TabPanel>
                <VStack align="stretch" spacing={4}>
                  <HStack>
                    <Badge colorScheme="yellow">JavaScript</Badge>
                    <Text fontSize="sm" color="gray.600">Quick to write, runs in V8 JavaScript engine</Text>
                  </HStack>

                  <Tabs variant="soft-rounded" size="sm">
                    <TabList flexWrap="wrap">
                      {Object.keys(jsExamples).map(eventType => (
                        <Tab key={eventType}>{eventType}</Tab>
                      ))}
                    </TabList>
                    <TabPanels>
                      {Object.entries(jsExamples).map(([eventType, code]) => (
                        <TabPanel key={eventType}>
                          <Box border="1px" borderColor="gray.200" borderRadius="md" overflow="hidden">
                            <CodeMirror
                              value={code}
                              extensions={[javascript()]}
                              theme={useColorModeValue('light', oneDark)}
                              readOnly
                              basicSetup={{
                                lineNumbers: true,
                                foldGutter: true,
                                highlightSelectionMatches: false
                              }}
                            />
                          </Box>
                        </TabPanel>
                      ))}
                    </TabPanels>
                  </Tabs>
                </VStack>
              </TabPanel>

              {/* Go Examples Tab */}
              <TabPanel>
                <VStack align="stretch" spacing={4}>
                  <HStack>
                    <Badge colorScheme="green">Go</Badge>
                    <Text fontSize="sm" color="gray.600">Compiled for better performance, hot-reload enabled</Text>
                  </HStack>

                  <Tabs variant="soft-rounded" size="sm">
                    <TabList flexWrap="wrap">
                      {Object.keys(goExamples).map(eventType => (
                        <Tab key={eventType}>{eventType}</Tab>
                      ))}
                    </TabList>
                    <TabPanels>
                      {Object.entries(goExamples).map(([eventType, code]) => (
                        <TabPanel key={eventType}>
                          <Box border="1px" borderColor="gray.200" borderRadius="md" overflow="hidden">
                            <CodeMirror
                              value={code}
                              extensions={[go()]}
                              theme={useColorModeValue('light', oneDark)}
                              readOnly
                              basicSetup={{
                                lineNumbers: true,
                                foldGutter: true,
                                highlightSelectionMatches: false
                              }}
                            />
                          </Box>
                        </TabPanel>
                      ))}
                    </TabPanels>
                  </Tabs>
                </VStack>
              </TabPanel>

              {/* API Reference Tab */}
              <TabPanel>
                <Tabs>
                  <TabList>
                    <Tab>JavaScript API</Tab>
                    <Tab>Go API</Tab>
                  </TabList>
                  <TabPanels>
                    {/* JavaScript API */}
                    <TabPanel>
                      <VStack align="stretch" spacing={6}>
                        <Box>
                          <Heading size="md" mb={3}>Available Objects</Heading>
                          <VStack align="stretch" spacing={2}>
                            {apiReference.js.objects.map(obj => (
                              <HStack key={obj.name} align="start" spacing={4}>
                                <Code fontSize="sm" colorScheme="blue" px={2} py={1}>{obj.name}</Code>
                                <Text fontSize="sm">{obj.desc}</Text>
                              </HStack>
                            ))}
                          </VStack>
                        </Box>

                        <Divider />

                        <Box>
                          <Heading size="md" mb={3}>Available Functions</Heading>
                          <VStack align="stretch" spacing={2}>
                            {apiReference.js.functions.map(func => (
                              <HStack key={func.name} align="start" spacing={4}>
                                <Code fontSize="sm" colorScheme="green" px={2} py={1}>{func.name}</Code>
                                <Text fontSize="sm">{func.desc}</Text>
                              </HStack>
                            ))}
                          </VStack>
                        </Box>
                      </VStack>
                    </TabPanel>

                    {/* Go API */}
                    <TabPanel>
                      <VStack align="stretch" spacing={6}>
                        <Box>
                          <Heading size="md" mb={3}>Available Objects</Heading>
                          <VStack align="stretch" spacing={2}>
                            {apiReference.go.objects.map(obj => (
                              <HStack key={obj.name} align="start" spacing={4}>
                                <Code fontSize="sm" colorScheme="blue" px={2} py={1}>{obj.name}</Code>
                                <Text fontSize="sm">{obj.desc}</Text>
                              </HStack>
                            ))}
                          </VStack>
                        </Box>

                        <Divider />

                        <Box>
                          <Heading size="md" mb={3}>Available Functions</Heading>
                          <VStack align="stretch" spacing={2}>
                            {apiReference.go.functions.map(func => (
                              <HStack key={func.name} align="start" spacing={4}>
                                <Code fontSize="sm" colorScheme="green" px={2} py={1}>{func.name}</Code>
                                <Text fontSize="sm">{func.desc}</Text>
                              </HStack>
                            ))}
                          </VStack>
                        </Box>
                      </VStack>
                    </TabPanel>
                  </TabPanels>
                </Tabs>
              </TabPanel>
            </TabPanels>
          </Tabs>
        </ModalBody>
      </ModalContent>
    </Modal>
  )
}

export default EventDocumentation