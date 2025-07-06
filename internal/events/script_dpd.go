package events

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	v8 "rogchap.com/v8go"
	"github.com/hjanuschka/go-deployd/internal/logging"
)

// setupDpdObject creates the dpd object for internal API access
func setupDpdObject(v8ctx *v8.Context, sc *ScriptContext) error {
	isolate := v8ctx.Isolate()
	
	// Get router from context if available
	if sc.ctx == nil || sc.ctx.HTTPHandler == nil {
		// No HTTP handler available, create empty dpd object
		dpdTemplate := v8.NewObjectTemplate(isolate)
		dpd, _ := dpdTemplate.NewInstance(v8ctx)
		v8ctx.Global().Set("dpd", dpd)
		return nil
	}
	
	httpRouter := sc.ctx.HTTPHandler
	
	// Create a factory function that creates collection proxies
	createCollectionFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		if len(args) == 0 || !args[0].IsString() {
			return v8.Undefined(isolate)
		}
		
		collectionName := args[0].String()
		return createCollectionProxy(v8ctx, sc, httpRouter.(http.Handler), collectionName)
	})
	
	// Set the factory function globally (temporarily)
	v8ctx.Global().Set("__createDpdCollection", createCollectionFunc.GetFunction(v8ctx))
	
	// Pass list of known collections if available
	collectionList := getAvailableCollections(sc)
	if len(collectionList) > 0 {
		// Convert to JavaScript array
		jsArray, _ := v8.NewValue(isolate, collectionList)
		v8ctx.Global().Set("__knownCollections", jsArray)
	}
	
	// Use JavaScript Proxy to create dynamic property access
	proxyScript := `
		// Create dpd object with dynamic collection access
		const dpd = new Proxy({}, {
			get: function(target, prop) {
				// Check if property already exists
				if (prop in target) {
					return target[prop];
				}
				
				// Skip symbol properties and internal properties
				if (typeof prop === 'symbol' || prop.startsWith('_')) {
					return undefined;
				}
				
				// Create collection proxy for any property access
				const collection = __createDpdCollection(prop);
				
				// Cache it for future access
				target[prop] = collection;
				
				return collection;
			}
		});
		
		// Pre-populate known collections if provided by the server
		// This helps with IDE autocomplete and console.log(dpd)
		if (typeof __knownCollections !== 'undefined' && Array.isArray(__knownCollections)) {
			__knownCollections.forEach(name => {
				if (typeof name === 'string') {
					dpd[name] = __createDpdCollection(name);
				}
			});
		}
		
		// Clean up the temporary functions
		delete globalThis.__createDpdCollection;
		delete globalThis.__knownCollections;
		
		// Return dpd for assignment
		dpd;
	`
	
	// Execute the proxy script
	result, err := v8ctx.RunScript(proxyScript, "dpd-proxy-setup")
	if err != nil {
		logging.Error("Failed to create dpd proxy", "js-dpd", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	
	// Set dpd as global
	v8ctx.Global().Set("dpd", result)
	
	// Also add to context for consistency
	contextValue, _ := v8ctx.Global().Get("context")
	if contextValue != nil && !contextValue.IsUndefined() {
		if contextObj, ok := contextValue.AsObject(); ok == nil {
			contextObj.Set("dpd", result)
		}
	}
	
	return nil
}

// createCollectionProxy creates a proxy object for a specific collection
func createCollectionProxy(v8ctx *v8.Context, sc *ScriptContext, httpRouter http.Handler, collectionName string) *v8.Value {
	isolate := v8ctx.Isolate()
	
	// Create collection proxy object
	collTemplate := v8.NewObjectTemplate(isolate)
	
	// Add collection name property for debugging
	collTemplate.Set("__collection", collectionName)
	
	// Add get method (list all or get by ID)
	getFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		var id string
		var query map[string]interface{}
		
		// First arg can be ID (string) or query (object)
		if len(args) > 0 {
			if args[0].IsString() {
				id = args[0].String()
			} else if args[0].IsObject() {
				queryJSON, _ := v8.JSONStringify(v8ctx, args[0])
				json.Unmarshal([]byte(queryJSON), &query)
			}
		}
		
		// Create promise for async operation
		resolver, err := v8.NewPromiseResolver(v8ctx)
		if err != nil {
			return v8.Undefined(isolate)
		}
		
		// Execute internal GET request synchronously
		api := NewInternalAPI(httpRouter, sc.ctx.Development)
		result, err := api.Collection(collectionName).Get(id, query)
		if err != nil {
			logging.Debug("dpd collection get error", "js-dpd", map[string]interface{}{
				"collection": collectionName,
				"error":      err.Error(),
			})
			// Reject promise with error
			errorVal, _ := v8.NewValue(isolate, err.Error())
			resolver.Reject(errorVal)
			return resolver.GetPromise().Value
		}
		
		// Convert result to V8 value
		resultJSON, _ := json.Marshal(result)
		v8Result, _ := v8.JSONParse(v8ctx, string(resultJSON))
		
		// Resolve promise with result
		resolver.Resolve(v8Result)
		return resolver.GetPromise().Value
	})
	collTemplate.Set("get", getFunc)
	
	// Add post method (create)
	postFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		var data interface{}
		
		if len(args) > 0 && args[0].IsObject() {
			dataJSON, _ := v8.JSONStringify(v8ctx, args[0])
			json.Unmarshal([]byte(dataJSON), &data)
		}
		
		// Create promise for async operation
		resolver, err := v8.NewPromiseResolver(v8ctx)
		if err != nil {
			return v8.Undefined(isolate)
		}
		
		// Execute internal POST request
		api := NewInternalAPI(httpRouter, sc.ctx.Development)
		result, err := api.Collection(collectionName).Post(data)
		if err != nil {
			logging.Debug("dpd collection post error", "js-dpd", map[string]interface{}{
				"collection": collectionName,
				"error":      err.Error(),
			})
			// Reject promise with error
			errorVal, _ := v8.NewValue(isolate, err.Error())
			resolver.Reject(errorVal)
			return resolver.GetPromise().Value
		}
		
		// Convert result to V8 value
		resultJSON, _ := json.Marshal(result)
		v8Result, _ := v8.JSONParse(v8ctx, string(resultJSON))
		
		// Resolve promise with result
		resolver.Resolve(v8Result)
		return resolver.GetPromise().Value
	})
	collTemplate.Set("post", postFunc)
	
	// Add put method (update)
	putFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		var id string
		var data interface{}
		
		if len(args) > 0 && args[0].IsString() {
			id = args[0].String()
		}
		if len(args) > 1 && args[1].IsObject() {
			dataJSON, _ := v8.JSONStringify(v8ctx, args[1])
			json.Unmarshal([]byte(dataJSON), &data)
		}
		
		// Create promise for async operation
		resolver, err := v8.NewPromiseResolver(v8ctx)
		if err != nil {
			return v8.Undefined(isolate)
		}
		
		// Execute internal PUT request
		api := NewInternalAPI(httpRouter, sc.ctx.Development)
		result, err := api.Collection(collectionName).Put(id, data)
		if err != nil {
			logging.Debug("dpd collection put error", "js-dpd", map[string]interface{}{
				"collection": collectionName,
				"error":      err.Error(),
			})
			// Reject promise with error
			errorVal, _ := v8.NewValue(isolate, err.Error())
			resolver.Reject(errorVal)
			return resolver.GetPromise().Value
		}
		
		// Convert result to V8 value
		resultJSON, _ := json.Marshal(result)
		v8Result, _ := v8.JSONParse(v8ctx, string(resultJSON))
		
		// Resolve promise with result
		resolver.Resolve(v8Result)
		return resolver.GetPromise().Value
	})
	collTemplate.Set("put", putFunc)
	
	// Add del method (delete)
	delFunc := v8.NewFunctionTemplate(isolate, func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		var id string
		
		if len(args) > 0 && args[0].IsString() {
			id = args[0].String()
		}
		
		// Create promise for async operation
		resolver, err := v8.NewPromiseResolver(v8ctx)
		if err != nil {
			return v8.Undefined(isolate)
		}
		
		// Execute internal DELETE request
		api := NewInternalAPI(httpRouter, sc.ctx.Development)
		_, err = api.Collection(collectionName).Delete(id)
		if err != nil {
			logging.Debug("dpd collection delete error", "js-dpd", map[string]interface{}{
				"collection": collectionName,
				"error":      err.Error(),
			})
			// Reject promise with error
			errorVal, _ := v8.NewValue(isolate, err.Error())
			resolver.Reject(errorVal)
			return resolver.GetPromise().Value
		}
		
		// Resolve promise with success
		resolver.Resolve(v8.Undefined(isolate))
		return resolver.GetPromise().Value
	})
	collTemplate.Set("del", delFunc)
	
	// Create and return collection instance
	collInstance, err := collTemplate.NewInstance(v8ctx)
	if err != nil {
		return v8.Undefined(isolate)
	}
	
	return collInstance.Value
}

// getAvailableCollections returns a list of available collection names
func getAvailableCollections(sc *ScriptContext) []string {
	// Try to get collections from the resource path
	collections := []string{}
	
	// If we have a resource, we know at least its collection exists
	if sc.ctx != nil && sc.ctx.Resource != nil {
		resourceName := sc.ctx.Resource.GetName()
		if resourceName != "" {
			collections = append(collections, resourceName)
		}
	}
	
	// Try to discover collections from the resources directory
	// This is a best-effort approach
	resourcesPath := "resources"
	if entries, err := os.ReadDir(resourcesPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				// Check if it has a config.json file (indicating it's a collection)
				configPath := filepath.Join(resourcesPath, entry.Name(), "config.json")
				if _, err := os.Stat(configPath); err == nil {
					name := entry.Name()
					// Skip special directories
					if !strings.HasPrefix(name, ".") && name != "metrics.json" {
						collections = append(collections, name)
					}
				}
			}
		}
	}
	
	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, col := range collections {
		if !seen[col] {
			seen[col] = true
			unique = append(unique, col)
		}
	}
	
	return unique
}