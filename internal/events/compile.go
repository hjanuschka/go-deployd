package events

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/dop251/goja"
	"github.com/hjanuschka/go-deployd/internal/context"
	"go.mongodb.org/mongo-driver/bson"
)

// CompileJS compiles JavaScript source code
func CompileJS(filename, source string) (*goja.Program, error) {
	return goja.Compile(filename, source, true)
}

// CompileGoPlugin compiles a Go source file to a plugin
func CompileGoPlugin(sourcePath, pluginPath string) error {
	// Read the source file
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Create a temporary wrapper file
	wrapperPath := strings.TrimSuffix(pluginPath, ".so") + "_wrapper.go"
	wrapper := createGoWrapper(string(source))
	
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0644); err != nil {
		return fmt.Errorf("failed to write wrapper: %w", err)
	}
	defer os.Remove(wrapperPath)

	// Create a temporary go.mod file for the plugin
	modPath := strings.TrimSuffix(pluginPath, ".so") + "_go.mod"
	modContent := `module plugin

go 1.21

require (
	github.com/hjanuschka/go-deployd v0.0.0
	go.mongodb.org/mongo-driver v1.13.0
)

replace github.com/hjanuschka/go-deployd => ` + getProjectRoot()
	
	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}
	defer os.Remove(modPath)

	// Compile the plugin
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", pluginPath, wrapperPath)
	cmd.Env = append(os.Environ(), 
		"GO111MODULE=on",
		"GOWORK=off", // Disable workspace mode
	)
	cmd.Dir = filepath.Dir(wrapperPath)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compilation failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// createGoWrapper creates a wrapper that implements the plugin interface
func createGoWrapper(userCode string) string {
	return fmt.Sprintf(`package main

import (
	"time"
	"strings"
	"github.com/hjanuschka/go-deployd/internal/events"
	"go.mongodb.org/mongo-driver/bson"
)

%s

// EventHandler is the exported plugin handler
var EventHandler eventHandler

type eventHandler struct{}

// Run implements the plugin interface
func (h eventHandler) Run(ctx *events.EventContext) error {
	return Run(ctx)
}
`, userCode)
}

// RunGoPlugin loads and executes a Go plugin
func RunGoPlugin(pluginPath string, ctx *context.Context, data bson.M) error {
	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Look up the EventHandler symbol
	symHandler, err := p.Lookup("EventHandler")
	if err != nil {
		return fmt.Errorf("EventHandler not found in plugin: %w", err)
	}

	// Create event context
	eventCtx := &EventContext{
		Ctx:      ctx,
		Data:     data,
		Errors:   make(map[string]string),
		Query:    ctx.Query,
		Internal: false, // TODO: Add Internal field to Context if needed
		IsRoot:   ctx.Session != nil && ctx.Session.IsRoot(),
	}

	if ctx.Session != nil {
		// TODO: Add User field to Session interface if needed
		// For now, try to get user from session data
		if user := ctx.Session.Get("user"); user != nil {
			if userMap, ok := user.(bson.M); ok {
				eventCtx.Me = userMap
			}
		}
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

	// Use reflection to call the Run method
	handler := symHandler.(interface{})
	if runnable, ok := handler.(interface{ Run(*EventContext) error }); ok {
		// Run with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					if r != "CANCEL" {
						cancelErr = fmt.Errorf("plugin panic: %v", r)
					}
				}
			}()
			err = runnable.Run(eventCtx)
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
	} else {
		return fmt.Errorf("plugin does not implement Run method")
	}

	return nil
}

// getProjectRoot attempts to find the project root directory
func getProjectRoot() string {
	// Try to find go.mod in parent directories
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}