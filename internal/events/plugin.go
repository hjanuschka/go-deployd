package events

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/hjanuschka/go-deployd/internal/context"
	"go.mongodb.org/mongo-driver/bson"
)

// EventPlugin represents a Go plugin for event handling
type EventPlugin interface {
	// Run executes the event logic
	Run(ctx *EventContext) error
}

// EventContext is defined in compile.go

// GoPluginManager manages Go plugin-based events
type GoPluginManager struct {
	plugins map[EventType]*plugin.Plugin
	mu      sync.RWMutex
}

// NewGoPluginManager creates a new Go plugin manager
func NewGoPluginManager() *GoPluginManager {
	return &GoPluginManager{
		plugins: make(map[EventType]*plugin.Plugin),
	}
}

// LoadPlugins loads all event plugins from the given config path
func (gpm *GoPluginManager) LoadPlugins(configPath string) error {
	eventFiles := map[EventType]string{
		EventGet:           "get.go",
		EventValidate:      "validate.go",
		EventPost:          "post.go",
		EventPut:           "put.go",
		EventDelete:        "delete.go",
		EventAfterCommit:   "aftercommit.go",
		EventBeforeRequest: "beforerequest.go",
	}

	gpm.mu.Lock()
	defer gpm.mu.Unlock()

	for eventType, filename := range eventFiles {
		sourcePath := filepath.Join(configPath, filename)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		// Compile the Go source to a plugin
		pluginPath := filepath.Join(configPath, ".plugins", strings.TrimSuffix(filename, ".go")+".so")
		if err := gpm.compilePlugin(sourcePath, pluginPath); err != nil {
			return fmt.Errorf("failed to compile %s: %w", filename, err)
		}

		// Load the compiled plugin
		p, err := plugin.Open(pluginPath)
		if err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", pluginPath, err)
		}

		gpm.plugins[eventType] = p
	}

	return nil
}

// compilePlugin compiles a Go source file to a plugin
func (gpm *GoPluginManager) compilePlugin(sourcePath, pluginPath string) error {
	// Ensure plugin directory exists
	pluginDir := filepath.Dir(pluginPath)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return err
	}

	// Create a wrapper that implements the EventPlugin interface
	wrapperPath := strings.TrimSuffix(sourcePath, ".go") + "_wrapper.go"
	if err := gpm.createWrapper(sourcePath, wrapperPath); err != nil {
		return err
	}
	defer os.Remove(wrapperPath)

	// Compile the plugin
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", pluginPath, wrapperPath)
	cmd.Env = append(os.Environ(), "GO111MODULE=on")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compilation failed: %w\n%s", err, output)
	}

	return nil
}

// createWrapper creates a wrapper file that implements EventPlugin
func (gpm *GoPluginManager) createWrapper(sourcePath, wrapperPath string) error {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	wrapper := fmt.Sprintf(`package main

import (
	"github.com/hjanuschka/go-deployd/internal/events"
	"go.mongodb.org/mongo-driver/bson"
)

// User-defined event code
%s

// Plugin is the exported plugin instance
var Plugin eventPlugin

type eventPlugin struct{}

// Run implements the EventPlugin interface
func (p eventPlugin) Run(ctx *events.EventContext) error {
	// Call the user-defined Run function
	return Run(ctx)
}
`, source)

	return os.WriteFile(wrapperPath, []byte(wrapper), 0644)
}

// GetPlugin returns a plugin for the given event type
func (gpm *GoPluginManager) GetPlugin(eventType EventType) (*plugin.Plugin, error) {
	gpm.mu.RLock()
	defer gpm.mu.RUnlock()

	p, exists := gpm.plugins[eventType]
	if !exists {
		return nil, nil
	}
	return p, nil
}

// RunPlugin executes a plugin with the given context
func (gpm *GoPluginManager) RunPlugin(eventType EventType, ctx *context.Context, data bson.M) error {
	p, err := gpm.GetPlugin(eventType)
	if err != nil || p == nil {
		return err
	}

	// Look up the plugin symbol
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin symbol not found: %w", err)
	}

	// Cast to EventPlugin interface
	eventPlugin, ok := symPlugin.(EventPlugin)
	if !ok {
		return fmt.Errorf("invalid plugin type")
	}

	// Create event context
	eventCtx := &EventContext{
		Ctx:      ctx,
		Data:     data,
		Errors:   make(map[string]string),
		Query:    ctx.Query,
		Internal: false, // TODO: Add Internal field to Context if needed
		IsRoot:   ctx.IsRoot,
	}

	if ctx.IsAuthenticated {
		// Create user data from JWT authentication
		userData := map[string]interface{}{
			"id":       ctx.UserID,
			"username": ctx.Username,
			"isRoot":   ctx.IsRoot,
		}
		eventCtx.Me = userData
	}

	// Set up cancel function
	var cancelErr error
	eventCtx.Cancel = func(message string, statusCode int) {
		cancelErr = &ScriptError{
			Message:    message,
			StatusCode: statusCode,
		}
		panic("CANCEL")
	}

	// Run the plugin with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				if r != "CANCEL" {
					cancelErr = fmt.Errorf("plugin panic: %v", r)
				}
			}
		}()
		err = eventPlugin.Run(eventCtx)
	}()

	if cancelErr != nil {
		return cancelErr
	}

	if err != nil {
		return err
	}

	if eventCtx.HasErrors() {
		return &ValidationError{Errors: eventCtx.Errors}
	}

	return nil
}
