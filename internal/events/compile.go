package events

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/hjanuschka/go-deployd/internal/context"
	"github.com/hjanuschka/go-deployd/internal/logging"
	"go.mongodb.org/mongo-driver/bson"
)

// EventContext provides context for event scripts (mirrored from plugin)
type EventContext struct {
	Ctx        *context.Context // For compatibility with existing code
	Data       map[string]interface{}
	Query      map[string]interface{}
	Me         map[string]interface{}
	Previous   map[string]interface{} // For PUT requests
	IsRoot     bool
	Internal   bool
	Errors     map[string]string
	Cancel     func(message string, statusCode int)
	hideFields []string
}

func (ctx *EventContext) HasErrors() bool {
	return len(ctx.Errors) > 0
}

func (ctx *EventContext) GetHiddenFields() []string {
	return ctx.hideFields
}

func (ctx *EventContext) Error(field, message string) {
	if ctx.Errors == nil {
		ctx.Errors = make(map[string]string)
	}
	ctx.Errors[field] = message
}

func (ctx *EventContext) Hide(field string) {
	ctx.hideFields = append(ctx.hideFields, field)
	delete(ctx.Data, field)
}

func (ctx *EventContext) Protect(field string) {
	delete(ctx.Data, field)
}

func (ctx *EventContext) IsMe(id string) bool {
	if ctx.Me != nil {
		if userID, ok := ctx.Me["id"].(string); ok {
			return userID == id
		}
	}
	return false
}

// CompileJS compiles JavaScript source code
func CompileJSLegacy(filename, source string) (*goja.Program, error) {
	return goja.Compile(filename, source, true)
}

// CompileGoPlugin compiles a Go source file to a plugin
func CompileGoPlugin(sourcePath, pluginPath string) error {
	// Read the source file
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Create a temporary directory for plugin compilation
	tempDir := filepath.Join(filepath.Dir(pluginPath), "temp_"+filepath.Base(strings.TrimSuffix(pluginPath, ".so")))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a temporary wrapper file in the temp directory
	wrapperPath := filepath.Join(tempDir, "main.go")
	wrapper := createGoWrapper(string(source))
	
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0644); err != nil {
		return fmt.Errorf("failed to write wrapper: %w", err)
	}

	// Create a temporary go.mod file for the plugin
	modPath := filepath.Join(tempDir, "go.mod")
	modContent := `module eventplugin

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/shopspring/decimal v1.4.0
	golang.org/x/crypto v0.39.0
)
`
	
	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Create a go.sum file with the required checksums
	sumPath := filepath.Join(tempDir, "go.sum")
	sumContent := `github.com/google/uuid v1.6.0 h1:NIvaJDMOsjHA8n1jAhLSgzrAzy1Hgr+hNrb57e+94F0=
github.com/google/uuid v1.6.0/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
github.com/shopspring/decimal v1.4.0 h1:bxl37RwXBklmTi0C79JfXCEBD1cqqHt0bbgBAGFp81k=
github.com/shopspring/decimal v1.4.0/go.mod h1:gawqmDU56v4yIKSwfBSFip1HdCCXN8/+DMd9qYNcwME=
golang.org/x/crypto v0.39.0 h1:SHs+kF4LP+f+p14esP5jAoDpHU8Gu/v9lFRK6IT5imM=
golang.org/x/crypto v0.39.0/go.mod h1:L+Xg3Wf6HoL4Bn4238Z6ft6KfEpN0tJGo53AAPC632U=
`
	
	if err := os.WriteFile(sumPath, []byte(sumContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.sum: %w", err)
	}

	// Get the Go root and construct the path to the Go executable
	goRoot := runtime.GOROOT()
	goExe := filepath.Join(goRoot, "bin", "go")
	
	// Fallback to PATH lookup if GOROOT doesn't work
	if _, err := os.Stat(goExe); err != nil {
		if goExe, err = exec.LookPath("go"); err != nil {
			return fmt.Errorf("failed to find go executable: %w", err)
		}
	}
	
	// Download dependencies first
	modCmd := exec.Command(goExe, "mod", "download")
	modCmd.Env = append(os.Environ(), 
		"GO111MODULE=on",
		"GOWORK=off",
	)
	modCmd.Dir = tempDir
	
	if modOutput, err := modCmd.CombinedOutput(); err != nil {
		// Log but don't fail - dependencies might already be available
		fmt.Printf("Go mod download output: %s\n", modOutput)
	}
	
	// Compile the plugin using the same Go version
	cmd := exec.Command(goExe, "build", "-buildmode=plugin", "-o", pluginPath, "main.go")
	cmd.Env = append(os.Environ(), 
		"GO111MODULE=on",
		"GOWORK=off", // Disable workspace mode
	)
	cmd.Dir = tempDir
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compilation failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// createGoWrapper is defined in compile_wrapper.go

// RunGoPlugin loads and executes a Go plugin
func RunGoPlugin(pluginPath string, ctx *context.Context, data bson.M) error {
	startTime := time.Now()
	
	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}
	
	loadTime := time.Since(startTime)

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
		Internal: false,
		IsRoot:   ctx.Session != nil && ctx.Session.IsRoot(),
	}

	if ctx.Session != nil {
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
	if runnable, ok := handler.(interface{ Run(interface{}) error }); ok {
		executeStart := time.Now()
		
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
		
		executeTime := time.Since(executeStart)
		
		// Log detailed timing
		logging.Debug("Go plugin execution details", "go-plugin", map[string]interface{}{
			"plugin":      filepath.Base(pluginPath),
			"loadTimeMs":  loadTime.Milliseconds(),
			"execTimeMs":  executeTime.Milliseconds(),
			"totalTimeMs": time.Since(startTime).Milliseconds(),
			"hasErrors":   eventCtx.HasErrors(),
			"cancelled":   cancelErr != nil,
		})

		if cancelErr != nil {
			return cancelErr
		}

		if err != nil {
			return err
		}

		if eventCtx.HasErrors() {
			return &ValidationError{Errors: eventCtx.Errors}
		}
		
		// Sync modified data back to the original data parameter
		for key, value := range eventCtx.Data {
			data[key] = value
		}
		
		// Apply hidden fields
		if hiddenFields := eventCtx.GetHiddenFields(); hiddenFields != nil {
			for _, field := range hiddenFields {
				delete(data, field)
			}
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
