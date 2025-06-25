package swagger

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hjanuschka/go-deployd/internal/resources"
)

// OpenAPISpec represents the OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    OpenAPIInfo            `json:"info"`
	Servers []OpenAPIServer        `json:"servers"`
	Paths   map[string]interface{} `json:"paths"`
	Components OpenAPIComponents   `json:"components"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type OpenAPIComponents struct {
	Schemas         map[string]interface{} `json:"schemas"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes"`
}

type OpenAPIPath struct {
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	OperationID string                 `json:"operationId,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Parameters  []interface{}          `json:"parameters,omitempty"`
	RequestBody interface{}            `json:"requestBody,omitempty"`
	Responses   map[string]interface{} `json:"responses"`
	Security    []map[string][]string  `json:"security,omitempty"`
}

// Generator generates OpenAPI specifications for collections
type Generator struct {
	baseURL     string
	collections []resources.Resource
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(baseURL string, collections []resources.Resource) *Generator {
	return &Generator{
		baseURL:     baseURL,
		collections: collections,
	}
}

// GenerateSpec generates the complete OpenAPI specification
func (g *Generator) GenerateSpec() (*OpenAPISpec, error) {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       "go-deployd API",
			Description: "RESTful API for go-deployd backend-as-a-service",
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{
				URL:         g.baseURL,
				Description: "go-deployd server",
			},
		},
		Paths: make(map[string]interface{}),
		Components: OpenAPIComponents{
			Schemas:         make(map[string]interface{}),
			SecuritySchemes: g.generateSecuritySchemes(),
		},
	}

	// Add authentication endpoints
	g.addAuthPaths(spec)

	// Add collection endpoints
	for _, collection := range g.collections {
		g.addCollectionPaths(spec, collection)
	}

	return spec, nil
}

// GenerateCollectionSpec generates OpenAPI spec for a specific collection
func (g *Generator) GenerateCollectionSpec(collection resources.Resource) (*OpenAPISpec, error) {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       fmt.Sprintf("%s API", strings.Title(collection.GetName())),
			Description: fmt.Sprintf("RESTful API for %s collection", collection.GetName()),
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{
				URL:         g.baseURL,
				Description: "go-deployd server",
			},
		},
		Paths: make(map[string]interface{}),
		Components: OpenAPIComponents{
			Schemas:         make(map[string]interface{}),
			SecuritySchemes: g.generateSecuritySchemes(),
		},
	}

	g.addCollectionPaths(spec, collection)
	return spec, nil
}

func (g *Generator) generateSecuritySchemes() map[string]interface{} {
	return map[string]interface{}{
		"BearerAuth": map[string]interface{}{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
			"description":  "JWT token obtained from /auth/login",
		},
		"MasterKey": map[string]interface{}{
			"type":        "apiKey",
			"in":          "header",
			"name":        "X-Master-Key",
			"description": "Master key for administrative access",
		},
	}
}

func (g *Generator) addAuthPaths(spec *OpenAPISpec) {
	// Login endpoint
	spec.Paths["/auth/login"] = map[string]interface{}{
		"post": OpenAPIPath{
			Summary:     "Authenticate user",
			Description: "Login with username/password or master key to get JWT token",
			OperationID: "login",
			Tags:        []string{"Authentication"},
			RequestBody: map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"username": map[string]interface{}{
									"type":        "string",
									"description": "Username for login",
								},
								"password": map[string]interface{}{
									"type":        "string",
									"description": "Password for login",
								},
								"masterKey": map[string]interface{}{
									"type":        "string",
									"description": "Master key for administrative access",
								},
							},
							"oneOf": []interface{}{
								map[string]interface{}{
									"required": []string{"username", "password"},
								},
								map[string]interface{}{
									"required": []string{"masterKey"},
								},
							},
						},
					},
				},
			},
			Responses: map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Login successful",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/LoginResponse",
							},
						},
					},
				},
				"401": map[string]interface{}{
					"description": "Authentication failed",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/Error",
							},
						},
					},
				},
			},
		},
	}

	// Token validation endpoint
	spec.Paths["/auth/validate"] = map[string]interface{}{
		"get": OpenAPIPath{
			Summary:     "Validate JWT token",
			Description: "Validate the provided JWT token",
			OperationID: "validateToken",
			Tags:        []string{"Authentication"},
			Security: []map[string][]string{
				{"BearerAuth": {}},
			},
			Responses: map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Token is valid",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/TokenValidation",
							},
						},
					},
				},
				"401": map[string]interface{}{
					"description": "Token is invalid or expired",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/Error",
							},
						},
					},
				},
			},
		},
	}

	// Add common schemas
	spec.Components.Schemas["LoginResponse"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"token": map[string]interface{}{
				"type":        "string",
				"description": "JWT token",
			},
			"expiresAt": map[string]interface{}{
				"type":        "integer",
				"description": "Token expiration timestamp",
			},
			"isRoot": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether user has root privileges",
			},
			"user": map[string]interface{}{
				"type":        "object",
				"description": "User information",
			},
		},
	}

	spec.Components.Schemas["TokenValidation"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"valid": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether token is valid",
			},
			"userID": map[string]interface{}{
				"type":        "string",
				"description": "User ID from token",
			},
			"username": map[string]interface{}{
				"type":        "string",
				"description": "Username from token",
			},
			"isRoot": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether user has root privileges",
			},
			"exp": map[string]interface{}{
				"type":        "integer",
				"description": "Token expiration timestamp",
			},
		},
	}

	spec.Components.Schemas["Error"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"error": map[string]interface{}{
				"type":        "string",
				"description": "Error message",
			},
		},
	}
}

func (g *Generator) addCollectionPaths(spec *OpenAPISpec, collection resources.Resource) {
	collectionName := collection.GetName()
	path := collection.GetPath()
	
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Generate schema for the collection
	schema := g.generateCollectionSchema(collection)
	spec.Components.Schemas[strings.Title(collectionName)] = schema

	// Security for all operations
	security := []map[string][]string{
		{"BearerAuth": {}},
		{"MasterKey": {}},
	}

	// Collection operations (list and create)
	spec.Paths[path] = map[string]interface{}{
		"get": OpenAPIPath{
			Summary:     fmt.Sprintf("List %s", collectionName),
			Description: fmt.Sprintf("Get all documents from %s collection", collectionName),
			OperationID: fmt.Sprintf("list%s", strings.Title(collectionName)),
			Tags:        []string{strings.Title(collectionName)},
			Parameters:  g.generateQueryParameters(),
			Security:    security,
			Responses: map[string]interface{}{
				"200": map[string]interface{}{
					"description": "List of documents",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"$ref": fmt.Sprintf("#/components/schemas/%s", strings.Title(collectionName)),
								},
							},
						},
					},
				},
			},
		},
		"post": OpenAPIPath{
			Summary:     fmt.Sprintf("Create %s", collectionName),
			Description: fmt.Sprintf("Create a new document in %s collection", collectionName),
			OperationID: fmt.Sprintf("create%s", strings.Title(collectionName)),
			Tags:        []string{strings.Title(collectionName)},
			RequestBody: map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/%s", strings.Title(collectionName)),
						},
					},
				},
			},
			Security: security,
			Responses: map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Document created successfully",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": fmt.Sprintf("#/components/schemas/%s", strings.Title(collectionName)),
							},
						},
					},
				},
			},
		},
	}

	// Document operations (get, update, delete)
	spec.Paths[path+"/{id}"] = map[string]interface{}{
		"get": OpenAPIPath{
			Summary:     fmt.Sprintf("Get %s by ID", collectionName),
			Description: fmt.Sprintf("Get a specific document from %s collection", collectionName),
			OperationID: fmt.Sprintf("get%sByID", strings.Title(collectionName)),
			Tags:        []string{strings.Title(collectionName)},
			Parameters: []interface{}{
				map[string]interface{}{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"description": "Document ID",
					"schema": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Security: security,
			Responses: map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Document found",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": fmt.Sprintf("#/components/schemas/%s", strings.Title(collectionName)),
							},
						},
					},
				},
				"404": map[string]interface{}{
					"description": "Document not found",
				},
			},
		},
		"put": OpenAPIPath{
			Summary:     fmt.Sprintf("Update %s", collectionName),
			Description: fmt.Sprintf("Update a document in %s collection", collectionName),
			OperationID: fmt.Sprintf("update%s", strings.Title(collectionName)),
			Tags:        []string{strings.Title(collectionName)},
			Parameters: []interface{}{
				map[string]interface{}{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"description": "Document ID",
					"schema": map[string]interface{}{
						"type": "string",
					},
				},
			},
			RequestBody: map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/%s", strings.Title(collectionName)),
						},
					},
				},
			},
			Security: security,
			Responses: map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Document updated successfully",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": fmt.Sprintf("#/components/schemas/%s", strings.Title(collectionName)),
							},
						},
					},
				},
			},
		},
		"delete": OpenAPIPath{
			Summary:     fmt.Sprintf("Delete %s", collectionName),
			Description: fmt.Sprintf("Delete a document from %s collection", collectionName),
			OperationID: fmt.Sprintf("delete%s", strings.Title(collectionName)),
			Tags:        []string{strings.Title(collectionName)},
			Parameters: []interface{}{
				map[string]interface{}{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"description": "Document ID",
					"schema": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Security: security,
			Responses: map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Document deleted successfully",
				},
				"404": map[string]interface{}{
					"description": "Document not found",
				},
			},
		},
	}
}

func (g *Generator) generateCollectionSchema(collection resources.Resource) map[string]interface{} {
	// Try to get collection config if it's a Collection type
	if coll, ok := collection.(*resources.Collection); ok {
		config := coll.GetConfig()
		if config != nil && config.Properties != nil {
			return g.generateSchemaFromProperties(config.Properties)
		}
	}

	// Default schema for collections without explicit configuration
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Unique document identifier",
				"readOnly":    true,
			},
			"createdAt": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Document creation timestamp",
				"readOnly":    true,
			},
			"updatedAt": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Document last update timestamp",
				"readOnly":    true,
			},
		},
		"additionalProperties": true,
	}
}

func (g *Generator) generateSchemaFromProperties(properties map[string]resources.Property) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	// Add ID field
	schemaProps := schema["properties"].(map[string]interface{})
	schemaProps["id"] = map[string]interface{}{
		"type":        "string",
		"description": "Unique document identifier",
		"readOnly":    true,
	}

	required := []string{}

	for name, prop := range properties {
		propSchema := g.generatePropertySchema(prop)
		schemaProps[name] = propSchema

		if prop.Required {
			required = append(required, name)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func (g *Generator) generatePropertySchema(prop resources.Property) map[string]interface{} {
	schema := map[string]interface{}{}

	switch prop.Type {
	case "string":
		schema["type"] = "string"
	case "number":
		schema["type"] = "number"
	case "boolean":
		schema["type"] = "boolean"
	case "date":
		schema["type"] = "string"
		schema["format"] = "date-time"
	case "array":
		schema["type"] = "array"
		schema["items"] = map[string]interface{}{"type": "string"}
	case "object":
		schema["type"] = "object"
		schema["additionalProperties"] = true
	default:
		schema["type"] = "string"
	}

	if prop.Default != nil {
		schema["default"] = prop.Default
	}

	return schema
}

func (g *Generator) generateQueryParameters() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"name":        "limit",
			"in":          "query",
			"description": "Maximum number of documents to return",
			"schema": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
				"maximum": 1000,
				"default": 100,
			},
		},
		map[string]interface{}{
			"name":        "skip",
			"in":          "query",
			"description": "Number of documents to skip",
			"schema": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
				"default": 0,
			},
		},
		map[string]interface{}{
			"name":        "sort",
			"in":          "query",
			"description": "Sort field and direction (e.g., 'name' or '-createdAt')",
			"schema": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

// ToJSON converts the OpenAPI spec to JSON
func (spec *OpenAPISpec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}