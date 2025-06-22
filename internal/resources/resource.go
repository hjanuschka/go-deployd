package resources

import (
	"github.com/hjanuschka/go-deployd/internal/context"
)

// Resource represents a deployable resource that can handle HTTP requests
type Resource interface {
	GetName() string
	GetPath() string
	Handle(ctx *context.Context) error
}

// Property defines a field in a collection schema
type Property struct {
	Type     string      `json:"type"`
	Required bool        `json:"required,omitempty"`
	Default  interface{} `json:"default,omitempty"`
}

// BaseResource provides common functionality for all resources
type BaseResource struct {
	name string
	path string
}

func NewBaseResource(name string) *BaseResource {
	return &BaseResource{
		name: name,
		path: "/" + name,
	}
}

func (r *BaseResource) GetName() string {
	return r.name
}

func (r *BaseResource) GetPath() string {
	return r.path
}

func (r *BaseResource) Handle(ctx *context.Context) error {
	return ctx.WriteError(501, "Not implemented")
}