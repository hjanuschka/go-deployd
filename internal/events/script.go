package events

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"go.mongodb.org/mongo-driver/bson"
)

// Script represents a JavaScript event script
type Script struct {
	source   string
	path     string
	compiled *goja.Program
	mu       sync.RWMutex
}

// EventType represents the type of event
type EventType string

const (
	EventGet           EventType = "Get"
	EventValidate      EventType = "Validate"
	EventPost          EventType = "Post"
	EventPut           EventType = "Put"
	EventDelete        EventType = "Delete"
	EventAfterCommit   EventType = "AfterCommit"
	EventBeforeRequest EventType = "BeforeRequest"
)

// ScriptManager manages event scripts for a collection
type ScriptManager struct {
	scripts map[EventType]*Script
	mu      sync.RWMutex
}

// NewScriptManager creates a new script manager
func NewScriptManager() *ScriptManager {
	return &ScriptManager{
		scripts: make(map[EventType]*Script),
	}
}

// LoadScripts loads all event scripts from the given config path
func (sm *ScriptManager) LoadScripts(configPath string) error {
	eventFiles := map[EventType]string{
		EventGet:           "get.js",
		EventValidate:      "validate.js",
		EventPost:          "post.js",
		EventPut:           "put.js",
		EventDelete:        "delete.js",
		EventAfterCommit:   "aftercommit.js",
		EventBeforeRequest: "beforerequest.js",
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for eventType, filename := range eventFiles {
		filePath := filepath.Join(configPath, filename)
		if content, err := os.ReadFile(filePath); err == nil {
			script := &Script{
				source: string(content),
				path:   filePath,
			}
			// Pre-compile the script
			if prog, err := goja.Compile(filePath, script.source, true); err == nil {
				script.compiled = prog
			}
			sm.scripts[eventType] = script
		}
	}

	return nil
}

// GetScript returns a script for the given event type
func (sm *ScriptManager) GetScript(eventType EventType) *Script {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.scripts[eventType]
}

// ScriptContext holds the execution context for a script
type ScriptContext struct {
	ctx        *context.Context
	data       bson.M
	errors     map[string]string
	cancelled  bool
	cancelMsg  string
	statusCode int
	vm         *goja.Runtime
}

// Run executes the script in the given context
func (s *Script) Run(ctx *context.Context, data bson.M) (*ScriptContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vm := goja.New()
	scriptCtx := &ScriptContext{
		ctx:    ctx,
		data:   data,
		errors: make(map[string]string),
		vm:     vm,
	}

	// Set up the script environment
	if err := scriptCtx.setupEnvironment(); err != nil {
		return nil, err
	}

	// Execute the script
	logging.Debug("Executing JavaScript script", "js-execution", map[string]interface{}{
		"hasCompiledScript": s.compiled != nil,
		"scriptSource":      s.source,
		"scriptPath":        s.path,
	})
	
	if s.compiled != nil {
		if _, err := vm.RunProgram(s.compiled); err != nil {
			logging.Debug("JavaScript execution failed (compiled)", "js-execution", map[string]interface{}{
				"error": err.Error(),
			})
			return scriptCtx, fmt.Errorf("script error: %w", err)
		}
	} else {
		if _, err := vm.RunString(s.source); err != nil {
			logging.Debug("JavaScript execution failed (source)", "js-execution", map[string]interface{}{
				"error": err.Error(),
			})
			return scriptCtx, fmt.Errorf("script error: %w", err)
		}
	}
	
	logging.Debug("JavaScript execution completed", "js-execution", map[string]interface{}{
		"hasCancelled": scriptCtx.cancelled,
		"hasErrors":    len(scriptCtx.errors) > 0,
		"errors":       scriptCtx.errors,
	})

	return scriptCtx, nil
}

// setupEnvironment sets up the JavaScript environment
func (sc *ScriptContext) setupEnvironment() error {
	vm := sc.vm

	// Debug logging for script context setup
	logging.Debug("Setting up JavaScript environment", "js-context", map[string]interface{}{
		"dataKeys":   getMapKeys(sc.data),
		"dataValues": sc.data,
		"hasData":    sc.data != nil,
		"dataLen":    len(sc.data),
	})

	// Set data object for structured access
	logging.Debug("Setting JavaScript data object", "js-context", map[string]interface{}{
		"dataKeys": getMapKeys(sc.data),
		"dataValues": sc.data,
	})
	
	// Set 'data' object for accessing fields like data.title
	vm.Set("data", sc.data)
	
	// Also set 'this' object for backward compatibility
	vm.Set("this", sc.data)

	// cancel() function
	vm.Set("cancel", func(msg string, statusCode ...int) {
		sc.cancelled = true
		sc.cancelMsg = msg
		if len(statusCode) > 0 {
			sc.statusCode = statusCode[0]
		} else {
			sc.statusCode = 400
		}
		panic(vm.ToValue("CANCEL"))
	})

	// cancelIf() function
	vm.Set("cancelIf", func(condition bool, msg string, statusCode ...int) {
		if condition {
			sc.cancelled = true
			sc.cancelMsg = msg
			if len(statusCode) > 0 {
				sc.statusCode = statusCode[0]
			} else {
				sc.statusCode = 400
			}
			panic(vm.ToValue("CANCEL"))
		}
	})

	// cancelUnless() function
	vm.Set("cancelUnless", func(condition bool, msg string, statusCode ...int) {
		if !condition {
			sc.cancelled = true
			sc.cancelMsg = msg
			if len(statusCode) > 0 {
				sc.statusCode = statusCode[0]
			} else {
				sc.statusCode = 400
			}
			panic(vm.ToValue("CANCEL"))
		}
	})

	// error() function
	vm.Set("error", func(field, message string) {
		sc.errors[field] = message
	})

	// hasErrors() function
	vm.Set("hasErrors", func() bool {
		return len(sc.errors) > 0
	})

	// me - current user
	var me interface{}
	if sc.ctx.Session != nil {
		if user := sc.ctx.Session.Get("user"); user != nil {
			me = user
		}
	}
	vm.Set("me", me)

	// isMe() function
	vm.Set("isMe", func(id string) bool {
		if sc.ctx.Session != nil {
			if user := sc.ctx.Session.Get("user"); user != nil {
				if userMap, ok := user.(bson.M); ok {
					if userID, ok := userMap["id"].(string); ok {
						return userID == id
					}
				}
			}
		}
		return false
	})

	// query - request query parameters
	vm.Set("query", sc.ctx.Query)

	// internal - whether request is internal
	vm.Set("internal", false) // TODO: Add Internal field to Context if needed

	// isRoot - whether user is root
	vm.Set("isRoot", sc.ctx.Session != nil && sc.ctx.Session.IsRoot())

	// emit() function (simplified version)
	vm.Set("emit", func(args ...interface{}) {
		// TODO: Implement real-time event emission
		// For now, just log
		fmt.Printf("emit: %v\n", args)
	})

	// dpd object for internal requests
	dpd := make(map[string]interface{})
	// TODO: Add collection proxies to dpd object
	vm.Set("dpd", dpd)

	// Basic require() function for common utilities
	vm.Set("require", func(module string) interface{} {
		switch module {
		case "crypto":
			// Provide basic crypto utilities using Go's crypto
			cryptoObj := make(map[string]interface{})
			cryptoObj["randomUUID"] = func() string {
				// Use Go's built-in random generation (simplified UUID-like)
				rand.Seed(time.Now().UnixNano())
				return fmt.Sprintf("%x-%x-%x-%x-%x", 
					rand.Int63()&0xffffffff,
					rand.Int63()&0xffff,
					rand.Int63()&0xffff,
					rand.Int63()&0xffff,
					rand.Int63()&0xffffffffffff)
			}
			cryptoObj["randomBytes"] = func(size int) []byte {
				bytes := make([]byte, size)
				rand.Seed(time.Now().UnixNano())
				for i := range bytes {
					bytes[i] = byte(rand.Int63() % 256)
				}
				return bytes
			}
			return cryptoObj
		case "util":
			// Provide utility functions
			utilObj := make(map[string]interface{})
			utilObj["isArray"] = func(obj interface{}) bool {
				_, ok := obj.([]interface{})
				return ok
			}
			utilObj["isObject"] = func(obj interface{}) bool {
				_, ok := obj.(map[string]interface{})
				return ok
			}
			return utilObj
		case "path":
			// Basic path utilities
			pathObj := make(map[string]interface{})
			pathObj["extname"] = func(filePath string) string {
				parts := strings.Split(filePath, ".")
				if len(parts) > 1 {
					return "." + parts[len(parts)-1]
				}
				return ""
			}
			pathObj["basename"] = func(filePath string) string {
				parts := strings.Split(filePath, "/")
				return parts[len(parts)-1]
			}
			return pathObj
		default:
			panic(vm.ToValue(fmt.Sprintf("Module '%s' not found", module)))
		}
	})

	// deployd object with logging functionality
	deployedObj := make(map[string]interface{})
	deployedObj["log"] = func(messageArgs ...interface{}) {
		var message string
		var data map[string]interface{}
		
		if len(messageArgs) == 0 {
			return
		}
		
		if len(messageArgs) >= 1 {
			if msg, ok := messageArgs[0].(string); ok {
				message = msg
			} else {
				message = fmt.Sprintf("%v", messageArgs[0])
			}
		}
		
		if len(messageArgs) >= 2 {
			if dataObj, ok := messageArgs[1].(map[string]interface{}); ok {
				data = dataObj
			} else {
				// Convert to map
				data = map[string]interface{}{
					"data": messageArgs[1],
				}
			}
		}
		
		// Determine source from script path or context
		source := "javascript"
		if sc.ctx != nil && sc.ctx.Resource != nil {
			source = fmt.Sprintf("js:%s", sc.ctx.Resource.GetName())
		}
		
		logging.Info(message, source, data)
	}
	vm.Set("deployd", deployedObj)

	// console.log function for debugging
	consoleObj := make(map[string]interface{})
	consoleObj["log"] = func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		message := fmt.Sprintf("JS Console: %v", args)
		logging.Debug(message, "js-console", nil)
		return goja.Undefined()
	}
	vm.Set("console", consoleObj)

	// protect() function
	vm.Set("protect", func(property string) {
		// Remove property from data
		delete(sc.data, property)
	})

	// hide() function
	vm.Set("hide", func(property string) {
		// Remove property from data (alias for protect)
		delete(sc.data, property)
	})

	// changed() function
	vm.Set("changed", func(property string) bool {
		// TODO: Implement change tracking
		return false
	})

	// previous object (for PUT requests)
	vm.Set("previous", make(map[string]interface{}))

	return nil
}

// IsCancelled returns whether the script execution was cancelled
func (sc *ScriptContext) IsCancelled() bool {
	return sc.cancelled
}

// GetError returns the cancellation error if any
func (sc *ScriptContext) GetError() error {
	if sc.cancelled {
		return &ScriptError{
			Message:    sc.cancelMsg,
			StatusCode: sc.statusCode,
		}
	}
	if len(sc.errors) > 0 {
		return &ValidationError{
			Errors: sc.errors,
		}
	}
	return nil
}

// ScriptError represents a script cancellation error
type ScriptError struct {
	Message    string
	StatusCode int
}

func (e *ScriptError) Error() string {
	return e.Message
}

// ValidationError represents validation errors from a script
type ValidationError struct {
	Errors map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Errors))
	for field, msg := range e.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}
	return "validation errors: " + strings.Join(parts, ", ")
}

// Helper function to get map keys for logging
func getMapKeys(data map[string]interface{}) []string {
	if data == nil {
		return nil
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}