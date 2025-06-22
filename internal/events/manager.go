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
	v8Pool           *V8Pool
	mu               sync.RWMutex
}

// ScriptType represents the type of script
type ScriptType string

const (
	ScriptTypeJS ScriptType = "js"
	ScriptTypeGo ScriptType = "go"
)

// EventConfiguration represents per-event runtime configuration
type EventConfiguration struct {
	Runtime string `json:"runtime"` // "js" or "go"
}

// CompiledGoScript represents a compiled Go script
type CompiledGoScript struct {
	SourcePath   string
	PluginPath   string
	LastModified int64
}

// NewUniversalScriptManager creates a manager that supports both JS and Go
func NewUniversalScriptManager() *UniversalScriptManager {
	// Initialize V8 pool for JavaScript events
	v8Pool := GetV8Pool()
	
	return &UniversalScriptManager{
		jsScripts:        make(map[EventType]*Script),
		goPlugins:        make(map[EventType]*CompiledGoScript),
		scriptTypes:      make(map[EventType]ScriptType),
		hotReloadManager: nil, // Will be initialized when needed
		v8Pool:           v8Pool,
	}
}

// LoadScriptsWithConfig loads event scripts with runtime configuration
func (usm *UniversalScriptManager) LoadScriptsWithConfig(configPath string, eventConfig map[string]EventConfiguration) error {
	usm.configPath = configPath
	
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
		// Get preferred runtime from config
		preferredRuntime := "go" // default to Go
		if config, exists := eventConfig[baseName]; exists && config.Runtime != "" {
			preferredRuntime = config.Runtime
		}
		
		// Load only the configured runtime - no fallback
		if preferredRuntime == "go" {
			// Only try Go script - compile to plugin on startup
			goPath := filepath.Join(configPath, baseName+".go")
			if _, err := os.ReadFile(goPath); err == nil {
				fmt.Printf("📦 Compiling Go event script: %s/%s.go\n", filepath.Base(configPath), baseName)
				// Compile Go script to plugin
				pluginPath := filepath.Join(configPath, ".plugins", baseName+".so")
				if err := CompileGoPlugin(goPath, pluginPath); err != nil {
					fmt.Printf("❌ Failed to compile Go script %s: %v\n", goPath, err)
					// Don't load this event script at all if Go compilation fails
				} else {
					fmt.Printf("✅ Successfully compiled Go event script: %s/%s.go\n", filepath.Base(configPath), baseName)
					usm.goPlugins[eventType] = &CompiledGoScript{
						SourcePath:   goPath,
						PluginPath:   pluginPath,
						LastModified: 0, // Not used for startup compilation
					}
					usm.scriptTypes[eventType] = ScriptTypeGo
				}
			}
			// If no .go file exists, that's fine - just don't load any script for this event
		} else {
			// Only try JavaScript
			jsPath := filepath.Join(configPath, baseName+".js")
			if content, err := os.ReadFile(jsPath); err == nil {
				script := &Script{
					source: string(content),
					path:   jsPath,
				}
				
				// Pre-compile the script in V8 pool for better performance
				if usm.v8Pool != nil {
					if precompileErr := usm.v8Pool.PrecompileScript(jsPath, string(content)); precompileErr != nil {
						// Log error but continue - fallback to runtime compilation
						fmt.Printf("Warning: Failed to precompile JavaScript %s: %v\n", jsPath, precompileErr)
					} else {
						// Mark script as compiled for optimized execution
						script.isPrecompiled = true
					}
				}
				
				usm.jsScripts[eventType] = script
				usm.scriptTypes[eventType] = ScriptTypeJS
			}
			// If no .js file exists, that's fine - just don't load any script for this event
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
		goScript := usm.goPlugins[eventType]
		usm.mu.RUnlock()
		// Use compiled plugin for Go scripts
		if goScript != nil {
			return RunGoPlugin(goScript.PluginPath, ctx, data)
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

// LoadScripts loads all event scripts from the given config path (backward compatibility)
func (usm *UniversalScriptManager) LoadScripts(configPath string) error {
	return usm.LoadScriptsWithConfig(configPath, make(map[string]EventConfiguration))
}

// ReloadScript reloads a specific event script
func (usm *UniversalScriptManager) ReloadScript(eventType EventType) error {
	// This would allow hot-reloading of scripts during development
	return usm.LoadScripts(usm.configPath)
}

// LoadHotReloadScript compiles and hot-loads a Go script
func (usm *UniversalScriptManager) LoadHotReloadScript(eventType EventType, source string) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()
	
	// Write source to the actual file location
	eventName := strings.ToLower(string(eventType))
	sourcePath := filepath.Join(usm.configPath, eventName+".go")
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		return err
	}
	
	// Compile to plugin
	pluginPath := filepath.Join(usm.configPath, ".plugins", eventName+".so")
	fmt.Printf("🔄 Hot-reloading Go event script: %s/%s.go\n", filepath.Base(usm.configPath), eventName)
	if err := CompileGoPlugin(sourcePath, pluginPath); err != nil {
		fmt.Printf("❌ Failed to hot-reload Go script: %v\n", err)
		return err
	}
	
	fmt.Printf("✅ Successfully hot-reloaded Go event script: %s/%s.go\n", filepath.Base(usm.configPath), eventName)
	
	// Update plugin reference
	usm.goPlugins[eventType] = &CompiledGoScript{
		SourcePath:   sourcePath,
		PluginPath:   pluginPath,
		LastModified: 0,
	}
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