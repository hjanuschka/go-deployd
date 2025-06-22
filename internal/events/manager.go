package events

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hjanuschka/go-deployd/internal/context"
	"go.mongodb.org/mongo-driver/bson"
)

// UniversalScriptManager manages both JavaScript and Go event scripts
type UniversalScriptManager struct {
	jsScripts        map[EventType]*Script
	goPlugins        map[EventType]*CompiledGoScript
	hotReloadManager *HotReloadGoManager
	scriptTypes      map[EventType]ScriptType
	configPath       string
	mu               sync.RWMutex
}

// ScriptType represents the type of script
type ScriptType string

const (
	ScriptTypeJS ScriptType = "js"
	ScriptTypeGo ScriptType = "go"
)

// CompiledGoScript represents a compiled Go script
type CompiledGoScript struct {
	SourcePath   string
	PluginPath   string
	LastModified int64
}

// NewUniversalScriptManager creates a manager that supports both JS and Go
func NewUniversalScriptManager() *UniversalScriptManager {
	return &UniversalScriptManager{
		jsScripts:        make(map[EventType]*Script),
		goPlugins:        make(map[EventType]*CompiledGoScript),
		scriptTypes:      make(map[EventType]ScriptType),
		hotReloadManager: nil, // Will be initialized when needed
	}
}

// LoadScripts loads all event scripts from the given config path
func (usm *UniversalScriptManager) LoadScripts(configPath string) error {
	usm.configPath = configPath
	
	// Initialize hot-reload manager if needed
	if usm.hotReloadManager == nil {
		usm.hotReloadManager = NewHotReloadGoManager(configPath)
	}
	
	eventNames := map[EventType]string{
		EventGet:           "get",
		EventValidate:      "validate",
		EventPost:          "post",
		EventPut:           "put",
		EventDelete:        "delete",
		EventAfterCommit:   "aftercommit",
		EventBeforeRequest: "beforerequest",
	}
	
	usm.mu.Lock()
	defer usm.mu.Unlock()
	
	// Create plugins directory
	pluginDir := filepath.Join(configPath, ".plugins")
	os.MkdirAll(pluginDir, 0755)
	
	for eventType, baseName := range eventNames {
		// Check for Go script first
		goPath := filepath.Join(configPath, baseName+".go")
		if content, err := os.ReadFile(goPath); err == nil {
			// Load Go script for hot-reloading
			if err := usm.hotReloadManager.LoadScript(eventType, string(content)); err != nil {
				fmt.Printf("Warning: Failed to load Go script %s: %v\n", goPath, err)
			} else {
				usm.scriptTypes[eventType] = ScriptTypeGo
				continue
			}
		}
		
		// Check for JavaScript script
		jsPath := filepath.Join(configPath, baseName+".js")
		if content, err := os.ReadFile(jsPath); err == nil {
			// Load JavaScript script
			script := &Script{
				source: string(content),
				path:   jsPath,
			}
			// Pre-compile if possible
			if prog, err := CompileJS(jsPath, script.source); err == nil {
				script.compiled = prog
			}
			usm.jsScripts[eventType] = script
			usm.scriptTypes[eventType] = ScriptTypeJS
		}
	}
	
	return nil
}

// loadGoScript compiles and loads a Go script
func (usm *UniversalScriptManager) loadGoScript(eventType EventType, sourcePath string, modTime int64) error {
	pluginName := strings.TrimSuffix(filepath.Base(sourcePath), ".go")
	pluginPath := filepath.Join(usm.configPath, ".plugins", pluginName+".so")
	
	// Check if we need to recompile
	needsCompile := true
	if existing, exists := usm.goPlugins[eventType]; exists {
		if pluginInfo, err := os.Stat(pluginPath); err == nil {
			if pluginInfo.ModTime().Unix() > modTime {
				needsCompile = false
			}
		}
		if existing.LastModified == modTime {
			needsCompile = false
		}
	}
	
	if needsCompile {
		if err := CompileGoPlugin(sourcePath, pluginPath); err != nil {
			return err
		}
	}
	
	usm.goPlugins[eventType] = &CompiledGoScript{
		SourcePath:   sourcePath,
		PluginPath:   pluginPath,
		LastModified: modTime,
	}
	
	return nil
}

// RunEvent executes an event script
func (usm *UniversalScriptManager) RunEvent(eventType EventType, ctx *context.Context, data bson.M) error {
	usm.mu.RLock()
	scriptType, exists := usm.scriptTypes[eventType]
	if !exists {
		usm.mu.RUnlock()
		return nil // No script for this event
	}
	
	switch scriptType {
	case ScriptTypeGo:
		usm.mu.RUnlock()
		// Use hot-reload manager for Go scripts
		if usm.hotReloadManager != nil {
			return usm.hotReloadManager.RunScript(eventType, ctx, data)
		}
		return nil
		
	case ScriptTypeJS:
		jsScript := usm.jsScripts[eventType]
		usm.mu.RUnlock()
		return usm.runJSScript(jsScript, ctx, data)
		
	default:
		usm.mu.RUnlock()
		return nil
	}
}

// runGoPlugin executes a Go plugin
func (usm *UniversalScriptManager) runGoPlugin(script *CompiledGoScript, ctx *context.Context, data bson.M) error {
	return RunGoPlugin(script.PluginPath, ctx, data)
}

// runJSScript executes a JavaScript script
func (usm *UniversalScriptManager) runJSScript(script *Script, ctx *context.Context, data bson.M) error {
	scriptCtx, err := script.Run(ctx, data)
	if err != nil {
		return err
	}
	return scriptCtx.GetError()
}

// GetScriptInfo returns information about loaded scripts
func (usm *UniversalScriptManager) GetScriptInfo() map[string]interface{} {
	usm.mu.RLock()
	defer usm.mu.RUnlock()
	
	info := make(map[string]interface{})
	
	for eventType, scriptType := range usm.scriptTypes {
		eventName := strings.ToLower(string(eventType))
		switch scriptType {
		case ScriptTypeGo:
			if script, exists := usm.goPlugins[eventType]; exists {
				info[eventName] = map[string]interface{}{
					"type":   "go",
					"path":   script.SourcePath,
					"plugin": script.PluginPath,
				}
			}
		case ScriptTypeJS:
			if script, exists := usm.jsScripts[eventType]; exists {
				info[eventName] = map[string]interface{}{
					"type": "js",
					"path": script.path,
				}
			}
		}
	}
	
	return info
}

// ReloadScript reloads a specific event script
func (usm *UniversalScriptManager) ReloadScript(eventType EventType) error {
	// This would allow hot-reloading of scripts during development
	return usm.LoadScripts(usm.configPath)
}

// LoadHotReloadScript loads a Go script for hot-reloading
func (usm *UniversalScriptManager) LoadHotReloadScript(eventType EventType, source string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()
	
	// Initialize hot-reload manager if needed
	if usm.hotReloadManager == nil {
		usm.hotReloadManager = NewHotReloadGoManager(usm.configPath)
	}
	
	// Load the script for hot-reloading
	if err := usm.hotReloadManager.LoadScript(eventType, source); err != nil {
		return err
	}
	
	// Update script type
	usm.scriptTypes[eventType] = ScriptTypeGo
	return nil
}

// GetHotReloadInfo returns hot-reload information
func (usm *UniversalScriptManager) GetHotReloadInfo() map[string]interface{} {
	if usm.hotReloadManager != nil {
		return usm.hotReloadManager.GetScriptInfo()
	}
	return make(map[string]interface{})
}