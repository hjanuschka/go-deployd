package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"go.mongodb.org/mongo-driver/bson"
	v8 "rogchap.com/v8go"
)

// Script represents a JavaScript event script using V8 (compatible with goja interface)
type Script struct {
	source        string
	path          string
	compiled      *v8.UnboundScript
	isPrecompiled bool // Indicates if script is precompiled in V8 pool
	mu            sync.RWMutex
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

// ScriptManager manages event scripts for a collection using V8 (compatible with goja interface)
type ScriptManager struct {
	scripts map[EventType]*Script
	mu      sync.RWMutex
}

// NewScriptManager creates a new V8 script manager
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

			// Pre-compilation is handled during execution for better error handling
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

// ScriptContext holds the execution context for a V8 script (compatible with goja interface)
type ScriptContext struct {
	ctx        *context.Context
	data       bson.M
	errors     map[string]string
	cancelled  bool
	cancelMsg  string
	statusCode int
}

// Run executes the script in the given context using V8 (compatible with goja interface)
func (s *Script) Run(ctx *context.Context, data bson.M) (*ScriptContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scriptCtx := &ScriptContext{
		ctx:    ctx,
		data:   data,
		errors: make(map[string]string),
	}

	// Use V8 pool if script is precompiled for better performance
	if s.isPrecompiled {
		return s.runWithPool(scriptCtx)
	}

	// Fallback to traditional method for non-precompiled scripts
	return s.runTraditional(scriptCtx)
}

// runWithPool executes the script using the V8 pool (optimized path)
func (s *Script) runWithPool(scriptCtx *ScriptContext) (*ScriptContext, error) {
	pool := GetV8Pool()
	if pool == nil {
		// Fallback to traditional execution if pool is not available
		return s.runTraditional(scriptCtx)
	}

	// Acquire a context from the pool with timeout
	acquireStart := time.Now()
	eventCtx, err := pool.AcquireContext(5 * time.Second)
	acquireTime := time.Since(acquireStart)
	if err != nil {
		logging.Debug("Failed to acquire V8 context from pool, falling back", "js-execution", map[string]interface{}{
			"error": err.Error(),
		})
		return s.runTraditional(scriptCtx)
	}
	defer pool.ReleaseContext(eventCtx)

	// Execute the precompiled script
	logging.Debug("Executing precompiled JavaScript script with V8 pool", "js-execution", map[string]interface{}{
		"scriptPath": s.path,
	})

	executeStart := time.Now()
	poolErr := pool.ExecuteScript(eventCtx, s.path, scriptCtx)
	executeTime := time.Since(executeStart)

	// Log detailed timing
	logging.Info("JavaScript execution timing", "js-timing", map[string]interface{}{
		"script":        filepath.Base(s.path),
		"pooled":        true,
		"execTimeMs":    executeTime.Milliseconds(),
		"acquireTimeMs": acquireTime.Milliseconds(),
		"totalTimeMs":   time.Since(acquireStart).Milliseconds(),
		"hasErrors":     len(scriptCtx.errors) > 0,
		"errorCount":    len(scriptCtx.errors),
	})

	if poolErr != nil {
		// Check if it's a cancellation (our custom exception)
		if strings.Contains(poolErr.Error(), "CANCEL") {
			logging.Debug("JavaScript execution cancelled (V8 pool)", "js-execution", map[string]interface{}{
				"cancelMsg": scriptCtx.cancelMsg,
			})
		} else {
			logging.Debug("JavaScript execution failed (V8 pool)", "js-execution", map[string]interface{}{
				"error": poolErr.Error(),
			})
			return scriptCtx, fmt.Errorf("script error: %w", poolErr)
		}
	}

	logging.Debug("JavaScript execution completed (V8 pool)", "js-execution", map[string]interface{}{
		"hasCancelled": scriptCtx.cancelled,
		"hasErrors":    len(scriptCtx.errors) > 0,
		"errors":       scriptCtx.errors,
	})

	return scriptCtx, nil
}

// runTraditional executes the script using traditional V8 method (fallback)
func (s *Script) runTraditional(scriptCtx *ScriptContext) (*ScriptContext, error) {
	startTime := time.Now()

	// Create a new isolate for each script execution to avoid conflicts
	isolate := v8.NewIsolate()
	defer isolate.Dispose()

	v8ctx := v8.NewContext(isolate)
	defer v8ctx.Close()

	setupTime := time.Since(startTime)

	// Set up the script environment
	if err := setupV8Environment(v8ctx, scriptCtx); err != nil {
		return nil, err
	}

	// Execute the script
	logging.Debug("Executing JavaScript script with V8 (traditional)", "js-execution", map[string]interface{}{
		"hasCompiledScript": s.compiled != nil,
		"scriptPath":        s.path,
	})

	executeStart := time.Now()
	var err error
	if s.compiled != nil {
		_, err = s.compiled.Run(v8ctx)
	} else {
		_, err = v8ctx.RunScript(s.source, s.path)
	}
	executeTime := time.Since(executeStart)

	if err != nil {
		// Check if it's a cancellation (our custom exception)
		if strings.Contains(err.Error(), "CANCEL") {
			// This is expected for cancel() calls
			logging.Debug("JavaScript execution cancelled (V8 traditional)", "js-execution", map[string]interface{}{
				"cancelMsg": scriptCtx.cancelMsg,
			})
		} else {
			logging.Debug("JavaScript execution failed (V8 traditional)", "js-execution", map[string]interface{}{
				"error": err.Error(),
			})
			return scriptCtx, fmt.Errorf("script error: %w", err)
		}
	}

	// Extract modified data back from JavaScript to Go
	if err := extractModifiedData(v8ctx, scriptCtx); err != nil {
		logging.Debug("Failed to extract modified data from V8", "js-execution", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Log detailed timing for traditional execution
	logging.Info("JavaScript execution timing", "js-timing", map[string]interface{}{
		"script":      filepath.Base(s.path),
		"pooled":      false,
		"setupTimeMs": setupTime.Milliseconds(),
		"execTimeMs":  executeTime.Milliseconds(),
		"totalTimeMs": time.Since(startTime).Milliseconds(),
		"hasErrors":   len(scriptCtx.errors) > 0,
		"errorCount":  len(scriptCtx.errors),
	})

	logging.Debug("JavaScript execution completed (V8 traditional)", "js-execution", map[string]interface{}{
		"hasCancelled": scriptCtx.cancelled,
		"hasErrors":    len(scriptCtx.errors) > 0,
		"errors":       scriptCtx.errors,
	})

	return scriptCtx, nil
}

// extractModifiedData extracts the modified data object from V8 back to Go
func extractModifiedData(v8ctx *v8.Context, sc *ScriptContext) error {
	// Get the modified data object from JavaScript
	dataValue, err := v8ctx.Global().Get("data")
	if err != nil || dataValue == nil {
		return err
	}

	// Convert back to JSON and then to Go map
	dataJSON, err := v8.JSONStringify(v8ctx, dataValue)
	if err != nil {
		return err
	}

	// Parse JSON back to bson.M
	var modifiedData bson.M
	if err := json.Unmarshal([]byte(dataJSON), &modifiedData); err != nil {
		return err
	}

	// Update the script context data
	sc.data = modifiedData

	return nil
}

// setupV8Environment sets up the JavaScript environment for V8
func setupV8Environment(v8ctx *v8.Context, sc *ScriptContext) error {
	// Debug logging for script context setup
	logging.Debug("Setting up JavaScript environment (V8)", "js-context", map[string]interface{}{
		"dataKeys": getMapKeys(sc.data),
		"hasData":  sc.data != nil,
		"dataLen":  len(sc.data),
	})

	// Convert bson.M to JavaScript object
	if err := setDataObject(v8ctx, sc.data); err != nil {
		return err
	}

	// Set up functions
	if err := setupCancelFunctions(v8ctx, sc); err != nil {
		return err
	}

	if err := setupValidationFunctions(v8ctx, sc); err != nil {
		return err
	}

	if err := setupUserFunctions(v8ctx, sc); err != nil {
		return err
	}

	if err := setupUtilityFunctions(v8ctx, sc); err != nil {
		return err
	}

	if err := setupRequireFunction(v8ctx, sc); err != nil {
		return err
	}

	return nil
}

// setDataObject converts bson.M to JavaScript and sets data/this
func setDataObject(v8ctx *v8.Context, data bson.M) error {
	dataJSON, _ := json.Marshal(data)
	dataValue, err := v8.JSONParse(v8ctx, string(dataJSON))
	if err != nil {
		return err
	}
	v8ctx.Global().Set("data", dataValue)
	v8ctx.Global().Set("this", dataValue)
	return nil
}

// setupCancelFunctions sets up cancel(), cancelIf(), cancelUnless()
func setupCancelFunctions(v8ctx *v8.Context, sc *ScriptContext) error {
	isolate := v8ctx.Isolate()

	// cancel() function
	cancelFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		msg := "Request cancelled"
		statusCode := 400

		if len(args) > 0 {
			msg = args[0].String()
		}
		if len(args) > 1 && args[1].IsNumber() {
			statusCode = int(args[1].Integer())
		}

		sc.cancelled = true
		sc.cancelMsg = msg
		sc.statusCode = statusCode

		// Throw an exception to stop execution
		exception, _ := v8.NewValue(isolate, "CANCEL")
		isolate.ThrowException(exception)
		return v8.Undefined(isolate)
	})
	v8ctx.Global().Set("cancel", cancelFunc.GetFunction(v8ctx))

	// cancelIf() function
	cancelIfFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) == 0 {
			return v8.Undefined(isolate)
		}

		condition := args[0].Boolean()
		if condition {
			msg := "Request cancelled"
			statusCode := 400

			if len(args) > 1 {
				msg = args[1].String()
			}
			if len(args) > 2 && args[2].IsNumber() {
				statusCode = int(args[2].Integer())
			}

			sc.cancelled = true
			sc.cancelMsg = msg
			sc.statusCode = statusCode

			exception, _ := v8.NewValue(isolate, "CANCEL")
			isolate.ThrowException(exception)
		}
		return v8.Undefined(isolate)
	})
	v8ctx.Global().Set("cancelIf", cancelIfFunc.GetFunction(v8ctx))

	// cancelUnless() function
	cancelUnlessFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) == 0 {
			return v8.Undefined(isolate)
		}

		condition := args[0].Boolean()
		if !condition {
			msg := "Request cancelled"
			statusCode := 400

			if len(args) > 1 {
				msg = args[1].String()
			}
			if len(args) > 2 && args[2].IsNumber() {
				statusCode = int(args[2].Integer())
			}

			sc.cancelled = true
			sc.cancelMsg = msg
			sc.statusCode = statusCode

			exception, _ := v8.NewValue(isolate, "CANCEL")
			isolate.ThrowException(exception)
		}
		return v8.Undefined(isolate)
	})
	v8ctx.Global().Set("cancelUnless", cancelUnlessFunc.GetFunction(v8ctx))

	return nil
}

// setupValidationFunctions sets up error(), hasErrors()
func setupValidationFunctions(v8ctx *v8.Context, sc *ScriptContext) error {
	isolate := v8ctx.Isolate()

	// error() function
	errorFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) >= 2 {
			field := args[0].String()
			message := args[1].String()
			sc.errors[field] = message
		}
		return v8.Undefined(isolate)
	})
	v8ctx.Global().Set("error", errorFunc.GetFunction(v8ctx))

	// hasErrors() function
	hasErrorsFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		result, _ := v8.NewValue(isolate, len(sc.errors) > 0)
		return result
	})
	v8ctx.Global().Set("hasErrors", hasErrorsFunc.GetFunction(v8ctx))

	return nil
}

// setupUserFunctions sets up me, isMe(), query, isRoot
func setupUserFunctions(v8ctx *v8.Context, sc *ScriptContext) error {
	isolate := v8ctx.Isolate()

	// me - current user
	var meValue *v8.Value
	if sc.ctx.IsAuthenticated {
		// Create user data from JWT authentication
		userData := map[string]interface{}{
			"id":       sc.ctx.UserID,
			"username": sc.ctx.Username,
			"isRoot":   sc.ctx.IsRoot,
		}
		userJSON, _ := json.Marshal(userData)
		meValue, _ = v8.JSONParse(v8ctx, string(userJSON))
	}
	if meValue == nil {
		meValue = v8.Null(isolate)
	}
	v8ctx.Global().Set("me", meValue)

	// isMe() function
	isMeFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) == 0 {
			result, _ := v8.NewValue(isolate, false)
			return result
		}

		id := args[0].String()
		if sc.ctx.IsAuthenticated {
			result, _ := v8.NewValue(isolate, sc.ctx.UserID == id)
			return result
		}
		result, _ := v8.NewValue(isolate, false)
		return result
	})
	v8ctx.Global().Set("isMe", isMeFunc.GetFunction(v8ctx))

	// query - request query parameters
	queryJSON, _ := json.Marshal(sc.ctx.Query)
	queryValue, err := v8.JSONParse(v8ctx, string(queryJSON))
	if err != nil {
		return err
	}
	v8ctx.Global().Set("query", queryValue)

	// internal and isRoot
	internalValue, _ := v8.NewValue(isolate, false)
	v8ctx.Global().Set("internal", internalValue)

	isRootValue, _ := v8.NewValue(isolate, sc.ctx.IsRoot)
	v8ctx.Global().Set("isRoot", isRootValue)

	return nil
}

// setupUtilityFunctions sets up emit(), dpd, console, protect(), hide(), changed(), previous
func setupUtilityFunctions(v8ctx *v8.Context, sc *ScriptContext) error {
	isolate := v8ctx.Isolate()

	// emit() function
	emitFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		argsSlice := make([]interface{}, len(args))
		for i, arg := range args {
			argsSlice[i] = arg.String()
		}
		fmt.Printf("emit: %v\n", argsSlice)
		return v8.Undefined(isolate)
	})
	v8ctx.Global().Set("emit", emitFunc.GetFunction(v8ctx))

	// dpd object
	dpdObj := v8.NewObjectTemplate(isolate)
	dpdValue, _ := dpdObj.NewInstance(v8ctx)
	v8ctx.Global().Set("dpd", dpdValue)

	// deployd object with logging
	deployedObjTemplate := v8.NewObjectTemplate(isolate)
	logFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) == 0 {
			return v8.Undefined(isolate)
		}

		message := args[0].String()
		var data map[string]interface{}

		if len(args) >= 2 && args[1].IsObject() {
			dataJSON, err := v8.JSONStringify(v8ctx, args[1])
			if err == nil {
				json.Unmarshal([]byte(dataJSON), &data)
			}
		}

		source := "javascript"
		if sc.ctx != nil && sc.ctx.Resource != nil {
			source = fmt.Sprintf("js:%s", sc.ctx.Resource.GetName())
		}

		logging.Info(message, source, data)
		return v8.Undefined(isolate)
	})
	deployedObjTemplate.Set("log", logFunc)
	deployedObj, _ := deployedObjTemplate.NewInstance(v8ctx)
	v8ctx.Global().Set("deployd", deployedObj)

	// console.log
	consoleObjTemplate := v8.NewObjectTemplate(isolate)
	consoleLogFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		argsSlice := make([]interface{}, len(args))
		for i, arg := range args {
			argsSlice[i] = arg.String()
		}
		message := fmt.Sprintf("JS Console: %v", argsSlice)
		logging.Debug(message, "js-console", nil)
		return v8.Undefined(isolate)
	})
	consoleObjTemplate.Set("log", consoleLogFunc)
	consoleObj, _ := consoleObjTemplate.NewInstance(v8ctx)
	v8ctx.Global().Set("console", consoleObj)

	// protect() and hide() functions
	protectFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) > 0 {
			property := args[0].String()
			delete(sc.data, property)
		}
		return v8.Undefined(isolate)
	})
	v8ctx.Global().Set("protect", protectFunc.GetFunction(v8ctx))
	v8ctx.Global().Set("hide", protectFunc.GetFunction(v8ctx))

	// changed() function
	changedFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		// TODO: Implement change tracking
		result, _ := v8.NewValue(isolate, false)
		return result
	})
	v8ctx.Global().Set("changed", changedFunc.GetFunction(v8ctx))

	// previous object
	previousObj := v8.NewObjectTemplate(isolate)
	previousValue, _ := previousObj.NewInstance(v8ctx)
	v8ctx.Global().Set("previous", previousValue)

	return nil
}

// setupRequireFunction sets up require() with built-in modules and npm support
func setupRequireFunction(v8ctx *v8.Context, sc *ScriptContext) error {
	isolate := v8ctx.Isolate()

	requireFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) == 0 {
			return v8.Undefined(isolate)
		}

		module := args[0].String()

		switch module {
		case "crypto":
			return createCryptoModule(v8ctx)
		case "util":
			return createUtilModule(v8ctx)
		case "path":
			return createPathModule(v8ctx)
		default:
			// Try to load from npm modules
			return loadNodeModule(v8ctx, module)
		}
	})
	v8ctx.Global().Set("require", requireFunc.GetFunction(v8ctx))

	return nil
}

// createCryptoModule creates the crypto module for V8
func createCryptoModule(v8ctx *v8.Context) *v8.Value {
	isolate := v8ctx.Isolate()
	cryptoTemplate := v8.NewObjectTemplate(isolate)

	// randomUUID function - simplified UUID-like generation
	randomUUIDFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		// Create a simple UUID-like string
		uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			0x12345678, 0x1234, 0x5678, 0x9abc, 0x123456789012)
		result, _ := v8.NewValue(isolate, uuid)
		return result
	})
	cryptoTemplate.Set("randomUUID", randomUUIDFunc)

	// randomBytes function
	randomBytesFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		size := 16
		if len(args) > 0 && args[0].IsNumber() {
			size = int(args[0].Integer())
		}

		// Create a hex string representing random bytes
		hexString := fmt.Sprintf("%0*x", size*2, 0x123456789abcdef0)
		if len(hexString) > size*2 {
			hexString = hexString[:size*2]
		}
		result, _ := v8.NewValue(isolate, hexString)
		return result
	})
	cryptoTemplate.Set("randomBytes", randomBytesFunc)

	cryptoObj, _ := cryptoTemplate.NewInstance(v8ctx)
	return cryptoObj.Value
}

// createUtilModule creates the util module for V8
func createUtilModule(v8ctx *v8.Context) *v8.Value {
	isolate := v8ctx.Isolate()
	utilTemplate := v8.NewObjectTemplate(isolate)

	// isArray function
	isArrayFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) > 0 {
			result, _ := v8.NewValue(isolate, args[0].IsArray())
			return result
		}
		result, _ := v8.NewValue(isolate, false)
		return result
	})
	utilTemplate.Set("isArray", isArrayFunc)

	// isObject function
	isObjectFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) > 0 {
			result, _ := v8.NewValue(isolate, args[0].IsObject() && !args[0].IsArray())
			return result
		}
		result, _ := v8.NewValue(isolate, false)
		return result
	})
	utilTemplate.Set("isObject", isObjectFunc)

	utilObj, _ := utilTemplate.NewInstance(v8ctx)
	return utilObj.Value
}

// createPathModule creates the path module for V8
func createPathModule(v8ctx *v8.Context) *v8.Value {
	isolate := v8ctx.Isolate()
	pathTemplate := v8.NewObjectTemplate(isolate)

	// extname function
	extnameFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) > 0 {
			filePath := args[0].String()
			parts := strings.Split(filePath, ".")
			if len(parts) > 1 {
				result, _ := v8.NewValue(isolate, "."+parts[len(parts)-1])
				return result
			}
		}
		result, _ := v8.NewValue(isolate, "")
		return result
	})
	pathTemplate.Set("extname", extnameFunc)

	// basename function
	basenameFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) > 0 {
			filePath := args[0].String()
			parts := strings.Split(filePath, "/")
			result, _ := v8.NewValue(isolate, parts[len(parts)-1])
			return result
		}
		result, _ := v8.NewValue(isolate, "")
		return result
	})
	pathTemplate.Set("basename", basenameFunc)

	pathObj, _ := pathTemplate.NewInstance(v8ctx)
	return pathObj.Value
}

// loadNodeModule loads npm modules from js-sandbox/node_modules
func loadNodeModule(v8ctx *v8.Context, module string) *v8.Value {
	isolate := v8ctx.Isolate()

	// Check for package.json in js-sandbox/node_modules/MODULE
	moduleDir := filepath.Join("js-sandbox", "node_modules", module)
	packageJSONPath := filepath.Join(moduleDir, "package.json")

	if _, err := os.Stat(packageJSONPath); err != nil {
		// Module not found
		logging.Debug("npm module not found", "js-require", map[string]interface{}{
			"module":     module,
			"searchPath": moduleDir,
		})
		return v8.Undefined(isolate)
	}

	// Read package.json to find main file
	packageJSON, err := os.ReadFile(packageJSONPath)
	if err != nil {
		logging.Debug("Failed to read package.json", "js-require", map[string]interface{}{
			"module": module,
			"error":  err.Error(),
		})
		return v8.Undefined(isolate)
	}

	var pkg struct {
		Main string `json:"main"`
	}
	if err := json.Unmarshal(packageJSON, &pkg); err != nil {
		logging.Debug("Failed to parse package.json", "js-require", map[string]interface{}{
			"module": module,
			"error":  err.Error(),
		})
		return v8.Undefined(isolate)
	}

	mainFile := pkg.Main
	if mainFile == "" {
		mainFile = "index.js"
	}

	// Load the main file
	mainPath := filepath.Join(moduleDir, mainFile)
	moduleCode, err := os.ReadFile(mainPath)
	if err != nil {
		logging.Debug("Failed to read module main file", "js-require", map[string]interface{}{
			"module":   module,
			"mainPath": mainPath,
			"error":    err.Error(),
		})
		return v8.Undefined(isolate)
	}

	// Create a new context for the module execution
	moduleCtx := v8.NewContext(isolate)
	defer moduleCtx.Close()

	// Set up minimal Node.js environment for the module
	exportsObj := v8.NewObjectTemplate(isolate)
	exports, _ := exportsObj.NewInstance(moduleCtx)
	moduleCtx.Global().Set("exports", exports)

	// Set up module object
	moduleObjTemplate := v8.NewObjectTemplate(isolate)
	moduleObjInstance, _ := moduleObjTemplate.NewInstance(moduleCtx)
	moduleObjInstance.Set("exports", exports)
	moduleCtx.Global().Set("module", moduleObjInstance)

	// Set up require function for nested dependencies
	requireFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) == 0 {
			return v8.Undefined(isolate)
		}

		depModule := args[0].String()
		// For now, only support direct dependencies, not nested requires
		logging.Debug("Nested require not fully supported", "js-require", map[string]interface{}{
			"parentModule": module,
			"dependency":   depModule,
		})
		return v8.Undefined(isolate)
	})
	moduleCtx.Global().Set("require", requireFunc.GetFunction(moduleCtx))

	// Execute the module code
	_, err = moduleCtx.RunScript(string(moduleCode), mainPath)
	if err != nil {
		logging.Debug("Failed to execute npm module", "js-require", map[string]interface{}{
			"module": module,
			"error":  err.Error(),
		})
		return v8.Undefined(isolate)
	}

	// Get the exports from the module
	exportsValue, err := moduleCtx.Global().Get("exports")
	if err != nil {
		logging.Debug("Failed to get module exports", "js-require", map[string]interface{}{
			"module": module,
			"error":  err.Error(),
		})
		return v8.Undefined(isolate)
	}

	// Convert exports to JSON and back to ensure it works in the main context
	exportsJSON, err := v8.JSONStringify(moduleCtx, exportsValue)
	if err != nil {
		logging.Debug("Failed to stringify module exports", "js-require", map[string]interface{}{
			"module": module,
			"error":  err.Error(),
		})
		return v8.Undefined(isolate)
	}

	// Parse back in the main context
	result, err := v8.JSONParse(v8ctx, exportsJSON)
	if err != nil {
		logging.Debug("Failed to parse module exports in main context", "js-require", map[string]interface{}{
			"module": module,
			"error":  err.Error(),
		})
		return v8.Undefined(isolate)
	}

	logging.Debug("Successfully loaded npm module", "js-require", map[string]interface{}{
		"module": module,
	})

	return result
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
