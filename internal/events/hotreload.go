package events

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"github.com/hjanuschka/go-deployd/internal/context"
	"go.mongodb.org/mongo-driver/bson"
)

// HotReloadGoManager manages hot-reloadable Go plugins
type HotReloadGoManager struct {
	plugins    map[EventType]*HotPlugin
	configPath string
	mu         sync.RWMutex
}

// HotPlugin represents a hot-reloadable Go plugin
type HotPlugin struct {
	SourcePath   string
	PluginPath   string
	Plugin       *plugin.Plugin
	LastModified time.Time
	LastCompiled time.Time
}

// NewHotReloadGoManager creates a new hot-reload manager
func NewHotReloadGoManager(configPath string) *HotReloadGoManager {
	return &HotReloadGoManager{
		plugins:    make(map[EventType]*HotPlugin),
		configPath: configPath,
	}
}

// LoadScript loads or reloads a Go script
func (hrm *HotReloadGoManager) LoadScript(eventType EventType, source string) error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	// Create source file
	sourcePath := filepath.Join(hrm.configPath, ".hotreload", strings.ToLower(string(eventType))+".go")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write source to file
	wrappedSource := hrm.wrapGoSource(source)
	if err := os.WriteFile(sourcePath, []byte(wrappedSource), 0644); err != nil {
		return fmt.Errorf("failed to write source: %w", err)
	}

	// Compile to plugin
	pluginPath := strings.TrimSuffix(sourcePath, ".go") + ".so"
	if err := hrm.compilePlugin(sourcePath, pluginPath); err != nil {
		return fmt.Errorf("failed to compile plugin: %w", err)
	}

	// Load plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Store plugin info
	hrm.plugins[eventType] = &HotPlugin{
		SourcePath:   sourcePath,
		PluginPath:   pluginPath,
		Plugin:       p,
		LastModified: time.Now(),
		LastCompiled: time.Now(),
	}

	return nil
}

// RunScript executes a hot-loaded Go script
func (hrm *HotReloadGoManager) RunScript(eventType EventType, ctx *context.Context, data bson.M) error {
	hrm.mu.RLock()
	hotPlugin, exists := hrm.plugins[eventType]
	hrm.mu.RUnlock()

	if !exists {
		return nil // No script loaded
	}

	// Look up the Run function
	symRun, err := hotPlugin.Plugin.Lookup("Run")
	if err != nil {
		return fmt.Errorf("Run function not found: %w", err)
	}

	// Cast to function
	runFunc, ok := symRun.(func(*EventContext) error)
	if !ok {
		return fmt.Errorf("invalid Run function signature")
	}

	// Create event context
	eventCtx := &EventContext{
		Ctx:      ctx,
		Data:     data,
		Errors:   make(map[string]string),
		Query:    ctx.Query,
		Internal: false,
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

	// Execute with panic recovery
	var execErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if r != "CANCEL" {
					execErr = fmt.Errorf("runtime panic: %v", r)
				}
			}
		}()
		execErr = runFunc(eventCtx)
	}()

	if cancelErr != nil {
		return cancelErr
	}

	if execErr != nil {
		return execErr
	}

	if eventCtx.HasErrors() {
		return &ValidationError{Errors: eventCtx.Errors}
	}

	return nil
}

// compilePlugin compiles a Go source file to a plugin
func (hrm *HotReloadGoManager) compilePlugin(sourcePath, pluginPath string) error {
	// Create go.mod for the plugin
	dir := filepath.Dir(sourcePath)
	modPath := filepath.Join(dir, "go.mod")

	modContent := fmt.Sprintf(`module hotreload

go 1.21

require (
	github.com/hjanuschka/go-deployd v0.0.0
	go.mongodb.org/mongo-driver v1.13.0
)

replace github.com/hjanuschka/go-deployd => %s
`, hrm.getProjectRoot())

	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Compile the plugin
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", pluginPath, sourcePath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "GOWORK=off")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compilation failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// wrapGoSource wraps user Go code in proper package structure
func (hrm *HotReloadGoManager) wrapGoSource(userCode string) string {
	// Check if it already has package declaration
	if strings.Contains(userCode, "package ") {
		return userCode
	}

	return fmt.Sprintf(`package main

import (
	"strings"
	"time"
	"github.com/hjanuschka/go-deployd/internal/events"
)

%s

// Exported Run function for plugin
func Run(ctx *events.EventContext) error {
	return run(ctx)
}

// Rename user function to avoid conflicts
func run(ctx *events.EventContext) error {
	return nil
}`, userCode)
}

// getProjectRoot finds the project root directory
func (hrm *HotReloadGoManager) getProjectRoot() string {
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

// GetScriptInfo returns information about loaded scripts
func (hrm *HotReloadGoManager) GetScriptInfo() map[string]interface{} {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	info := make(map[string]interface{})
	for eventType, hotPlugin := range hrm.plugins {
		info[strings.ToLower(string(eventType))] = map[string]interface{}{
			"type":         "go-hotreload",
			"sourcePath":   hotPlugin.SourcePath,
			"pluginPath":   hotPlugin.PluginPath,
			"lastModified": hotPlugin.LastModified,
			"lastCompiled": hotPlugin.LastCompiled,
		}
	}
	return info
}

// ReloadAllIfChanged checks for changes and reloads if necessary
func (hrm *HotReloadGoManager) ReloadAllIfChanged() error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	for eventType, hotPlugin := range hrm.plugins {
		if stat, err := os.Stat(hotPlugin.SourcePath); err == nil {
			if stat.ModTime().After(hotPlugin.LastCompiled) {
				// Source has changed, recompile
				if err := hrm.compilePlugin(hotPlugin.SourcePath, hotPlugin.PluginPath); err != nil {
					fmt.Printf("Failed to recompile %s: %v\n", eventType, err)
					continue
				}

				// Reload plugin
				p, err := plugin.Open(hotPlugin.PluginPath)
				if err != nil {
					fmt.Printf("Failed to reload plugin %s: %v\n", eventType, err)
					continue
				}

				hotPlugin.Plugin = p
				hotPlugin.LastCompiled = time.Now()
				fmt.Printf("Hot-reloaded %s plugin\n", eventType)
			}
		}
	}

	return nil
}
