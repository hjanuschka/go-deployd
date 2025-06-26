package context

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/hjanuschka/go-deployd/internal/sessions"
)

type Context struct {
	Request         *http.Request
	Response        http.ResponseWriter
	Resource        Resource
	Session         *sessions.Session
	SessionStore    *sessions.SessionStore
	Router          Router
	URL             string
	Query           map[string]interface{}
	Body            map[string]interface{}
	Method          string
	Development     bool
	UserID          string
	Username        string
	IsRoot          bool
	IsAuthenticated bool
	ctx             context.Context
}

type Resource interface {
	GetName() string
	GetPath() string
}

type Router interface {
	Route(ctx *Context) error
}

type AuthData struct {
	UserID          string
	Username        string
	IsRoot          bool
	IsAuthenticated bool
}

func New(req *http.Request, res http.ResponseWriter, resource Resource, auth *AuthData, session *sessions.Session, sessionStore *sessions.SessionStore) *Context {
	ctx := &Context{
		Request:         req,
		Response:        res,
		Resource:        resource,
		Session:         session,
		SessionStore:    sessionStore,
		Method:          req.Method,
		ctx:             req.Context(),
	}

	if auth != nil {
		ctx.UserID = auth.UserID
		ctx.Username = auth.Username
		ctx.IsRoot = auth.IsRoot
		ctx.IsAuthenticated = auth.IsAuthenticated
	}

	ctx.parseURL()
	ctx.parseQuery()
	ctx.parseBody()

	return ctx
}

func (c *Context) parseURL() {
	c.URL = c.Request.URL.Path
	if c.Resource != nil {
		resourcePath := c.Resource.GetPath()
		if strings.HasPrefix(c.URL, resourcePath) {
			c.URL = strings.TrimPrefix(c.URL, resourcePath)
		}
	}
	if c.URL == "" {
		c.URL = "/"
	}
}

func (c *Context) parseQuery() {
	c.Query = make(map[string]interface{})

	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			value := values[0]

			if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
				var jsonValue interface{}
				if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
					c.Query[key] = jsonValue
					continue
				}
			}

			if num, err := strconv.ParseFloat(value, 64); err == nil {
				c.Query[key] = num
				continue
			}

			if value == "true" {
				c.Query[key] = true
				continue
			}
			if value == "false" {
				c.Query[key] = false
				continue
			}

			c.Query[key] = value
		} else {
			var convertedValues []interface{}
			for _, v := range values {
				convertedValues = append(convertedValues, v)
			}
			c.Query[key] = convertedValues
		}
	}
}

func (c *Context) parseBody() {
	c.Body = make(map[string]interface{})

	if c.Request.Body == nil {
		return
	}

	contentType := c.Request.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var jsonBody map[string]interface{}
		err := json.NewDecoder(c.Request.Body).Decode(&jsonBody)
		if err == nil {
			for k, v := range jsonBody {
				c.Body[k] = v
			}
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := c.Request.ParseForm(); err == nil {
			for key, values := range c.Request.PostForm {
				if len(values) == 1 {
					c.Body[key] = values[0]
				} else {
					c.Body[key] = values
				}
			}
		}
	}
}

func (c *Context) ParseJSON(v interface{}) error {
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func (c *Context) WriteJSON(data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(c.Response).Encode(data)
}

func (c *Context) WriteError(statusCode int, message string) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(statusCode)
	return json.NewEncoder(c.Response).Encode(map[string]interface{}{
		"error":   true,
		"message": message,
		"status":  statusCode,
	})
}

func (c *Context) GetID() string {
	if c.URL != "/" {
		parts := strings.Split(strings.Trim(c.URL, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	if id, exists := c.Query["id"]; exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}

	if id, exists := c.Body["id"]; exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}

	return ""
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) Done(err error, result interface{}) {
	if err != nil {
		c.WriteError(500, err.Error())
		return
	}

	if result != nil {
		c.WriteJSON(result)
	} else {
		c.Response.WriteHeader(204) // No Content
	}
}
