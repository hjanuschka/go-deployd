package events

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hjanuschka/go-deployd/internal/logging"
	v8 "rogchap.com/v8go"
)

// V8Pool manages a pool of pre-loaded V8 isolates and contexts for JavaScript events
type V8Pool struct {
	mu        sync.RWMutex
	isolates  []*v8.Isolate
	contexts  []*V8EventContext
	available chan *V8EventContext
	scripts   map[string]string // Source code by file path for per-isolate compilation
	compiled  map[string]map[*v8.Isolate]*v8.UnboundScript // Per-isolate compiled scripts
	poolSize  int
	isShutdown bool
}

// V8EventContext wraps a V8 context with its isolate for reuse
type V8EventContext struct {
	isolate    *v8.Isolate
	context    *v8.Context
	inUse      bool
	lastUsed   time.Time
	execCount  int  // Track number of executions to detect when to refresh
}

var (
	globalV8Pool *V8Pool
	poolOnce     sync.Once
)

// GetV8Pool returns the global V8 pool singleton
func GetV8Pool() *V8Pool {
	poolOnce.Do(func() {
		globalV8Pool = NewV8Pool(4) // Default pool size of 4 contexts
	})
	return globalV8Pool
}

// NewV8Pool creates a new V8 pool with the specified number of contexts
func NewV8Pool(poolSize int) *V8Pool {
	if poolSize <= 0 {
		poolSize = 4
	}
	
	pool := &V8Pool{
		isolates:  make([]*v8.Isolate, 0, poolSize),
		contexts:  make([]*V8EventContext, 0, poolSize),
		available: make(chan *V8EventContext, poolSize),
		scripts:   make(map[string]string),
		compiled:  make(map[string]map[*v8.Isolate]*v8.UnboundScript),
		poolSize:  poolSize,
	}
	
	// Initialize the pool
	if err := pool.initialize(); err != nil {
		logging.Error("Failed to initialize V8 pool", "v8-pool", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}
	
	logging.Info("V8 pool initialized successfully", "v8-pool", map[string]interface{}{
		"poolSize": poolSize,
	})
	
	return pool
}

// initialize creates and prepares all V8 contexts in the pool
func (pool *V8Pool) initialize() error {
	for i := 0; i < pool.poolSize; i++ {
		isolate := v8.NewIsolate()
		context := v8.NewContext(isolate)
		
		eventCtx := &V8EventContext{
			isolate:  isolate,
			context:  context,
			lastUsed: time.Now(),
		}
		
		pool.isolates = append(pool.isolates, isolate)
		pool.contexts = append(pool.contexts, eventCtx)
		pool.available <- eventCtx
	}
	
	return nil
}

// PrecompileScript stores JavaScript source for per-isolate compilation
func (pool *V8Pool) PrecompileScript(filePath, source string) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	
	if pool.isShutdown {
		return fmt.Errorf("V8 pool is shut down")
	}
	
	// Store source code for per-isolate compilation
	pool.scripts[filePath] = source
	
	// Initialize compiled map for this script
	if pool.compiled[filePath] == nil {
		pool.compiled[filePath] = make(map[*v8.Isolate]*v8.UnboundScript)
	}
	
	logging.Debug("JavaScript source stored for compilation", "v8-pool", map[string]interface{}{
		"filePath": filePath,
	})
	
	return nil
}

// LoadScriptsFromDirectory scans a directory and precompiles all JavaScript files
func (pool *V8Pool) LoadScriptsFromDirectory(dir string) error {
	if pool.isShutdown {
		return fmt.Errorf("V8 pool is shut down")
	}
	
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if filepath.Ext(path) == ".js" {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				logging.Error("Failed to read JavaScript file", "v8-pool", map[string]interface{}{
					"error": readErr.Error(),
					"path":  path,
				})
				return readErr
			}
			
			if compileErr := pool.PrecompileScript(path, string(content)); compileErr != nil {
				logging.Error("Failed to precompile JavaScript file", "v8-pool", map[string]interface{}{
					"error": compileErr.Error(),
					"path":  path,
				})
				return compileErr
			}
		}
		
		return nil
	})
}

// AcquireContext gets an available V8 context from the pool
func (pool *V8Pool) AcquireContext(timeout time.Duration) (*V8EventContext, error) {
	if pool.isShutdown {
		return nil, fmt.Errorf("V8 pool is shut down")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	select {
	case eventCtx := <-pool.available:
		eventCtx.inUse = true
		eventCtx.lastUsed = time.Now()
		return eventCtx, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for available V8 context")
	}
}

// ReleaseContext returns a V8 context to the pool for reuse
func (pool *V8Pool) ReleaseContext(eventCtx *V8EventContext) {
	if pool.isShutdown {
		return
	}
	
	eventCtx.execCount++
	
	// If context has been used too many times, recreate it to avoid pollution
	if eventCtx.execCount > 1000 {
		pool.recreateContext(eventCtx)
	} else {
		// Reset the context for reuse (less needed now with function wrapping)
		pool.resetContext(eventCtx)
	}
	
	eventCtx.inUse = false
	eventCtx.lastUsed = time.Now()
	
	// Return to pool
	select {
	case pool.available <- eventCtx:
		// Successfully returned to pool
	default:
		// Pool is full (shouldn't happen), log warning
		logging.Error("V8 pool overflow - context not returned", "v8-pool", map[string]interface{}{
			"poolSize": pool.poolSize,
		})
	}
}

// recreateContext creates a fresh context to avoid global pollution
func (pool *V8Pool) recreateContext(eventCtx *V8EventContext) {
	// Close old context
	if eventCtx.context != nil {
		eventCtx.context.Close()
	}
	
	// Create new context with same isolate
	eventCtx.context = v8.NewContext(eventCtx.isolate)
	eventCtx.execCount = 0
	
	logging.Debug("Recreated V8 context to avoid pollution", "v8-pool", map[string]interface{}{
		"reason": "execCount > 100",
	})
}

// resetContext clears the context state for reuse
func (pool *V8Pool) resetContext(eventCtx *V8EventContext) {
	// Get all global property names to clear user-defined variables
	globalObj := eventCtx.context.Global()
	
	// Clear known deployd globals
	deployGlobals := []string{
		"data", "query", "me", "previous", "isRoot", "internal", "errors", 
		"cancelled", "error", "hide", "protect", "cancel", "isMe",
	}
	for _, global := range deployGlobals {
		globalObj.Delete(global)
	}
	
	// Clear common npm module globals that might be declared
	npmGlobals := []string{
		"crypto", "bcrypt", "uuid", "lodash", "_", "moment", "axios", 
		"fetch", "Buffer", "process", "require", "module", "exports",
		"nodemailer", "validator", "jsonwebtoken", "jwt", "fs", "path",
	}
	for _, global := range npmGlobals {
		globalObj.Delete(global)
	}
	
	// Try to get property names and clear any non-standard globals
	// Note: V8Go doesn't expose PropertyNames, so we clear known problematic ones
	
	// Reset core state
	globalObj.Set("cancelled", false)
}

// ExecuteScript executes a script in the given context (compiles per-isolate if needed)
func (pool *V8Pool) ExecuteScript(eventCtx *V8EventContext, filePath string, scriptCtx *ScriptContext) error {
	pool.mu.Lock()
	source, exists := pool.scripts[filePath]
	if !exists {
		pool.mu.Unlock()
		return fmt.Errorf("script not found in pool: %s", filePath)
	}
	
	// Wrap script in function to avoid global variable pollution
	wrappedSource := pool.wrapScriptInFunction(source)
	
	// Check if wrapped script is already compiled for this isolate
	wrappedKey := filePath + "_wrapped"
	var compiled *v8.UnboundScript
	if pool.compiled[wrappedKey] != nil {
		compiled = pool.compiled[wrappedKey][eventCtx.isolate]
	}
	
	// Compile for this isolate if not already done
	if compiled == nil {
		var err error
		compiled, err = eventCtx.isolate.CompileUnboundScript(wrappedSource, filePath, v8.CompileOptions{})
		if err != nil {
			pool.mu.Unlock()
			return fmt.Errorf("failed to compile wrapped script for isolate: %w", err)
		}
		
		// Store compiled script for this isolate
		if pool.compiled[wrappedKey] == nil {
			pool.compiled[wrappedKey] = make(map[*v8.Isolate]*v8.UnboundScript)
		}
		pool.compiled[wrappedKey][eventCtx.isolate] = compiled
	}
	pool.mu.Unlock()
	
	// Set up the script environment in the context
	if err := setupV8Environment(eventCtx.context, scriptCtx); err != nil {
		return err
	}
	
	// Execute the compiled script
	_, err := compiled.Run(eventCtx.context)
	if err != nil {
		return err
	}
	
	// Extract modified data back from JavaScript
	return extractModifiedData(eventCtx.context, scriptCtx)
}

// wrapScriptInFunction wraps the script code in an IIFE to avoid global variable pollution
func (pool *V8Pool) wrapScriptInFunction(source string) string {
	return fmt.Sprintf(`
(function() {
	%s
})();
`, source)
}

// GetStats returns statistics about the V8 pool
func (pool *V8Pool) GetStats() map[string]interface{} {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	
	available := len(pool.available)
	inUse := pool.poolSize - available
	
	return map[string]interface{}{
		"poolSize":         pool.poolSize,
		"available":        available,
		"inUse":           inUse,
		"precompiledScripts": len(pool.scripts),
		"isShutdown":      pool.isShutdown,
	}
}

// Shutdown gracefully shuts down the V8 pool
func (pool *V8Pool) Shutdown() {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	
	if pool.isShutdown {
		return
	}
	
	pool.isShutdown = true
	close(pool.available)
	
	// Dispose of all contexts and isolates
	for _, eventCtx := range pool.contexts {
		if eventCtx.context != nil {
			eventCtx.context.Close()
		}
	}
	
	for _, isolate := range pool.isolates {
		if isolate != nil {
			isolate.Dispose()
		}
	}
	
	logging.Info("V8 pool shut down successfully", "v8-pool", nil)
}

// HasPrecompiledScript checks if a script is already precompiled
func (pool *V8Pool) HasPrecompiledScript(filePath string) bool {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	
	_, exists := pool.scripts[filePath]
	return exists
}

// RemovePrecompiledScript removes a precompiled script (for hot reloading)
func (pool *V8Pool) RemovePrecompiledScript(filePath string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	
	delete(pool.scripts, filePath)
	
	logging.Debug("Removed precompiled script", "v8-pool", map[string]interface{}{
		"filePath": filePath,
	})
}