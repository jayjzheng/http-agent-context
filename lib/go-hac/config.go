package hac

import (
	"net/http"
	"sync"
)

// PathResolver extracts the route pattern from a request. The middleware uses
// this to look up the registered RouteConfig for the request.
type PathResolver func(r *http.Request) string

// StdlibPathResolver uses the Go 1.22+ r.Pattern field to resolve the route
// pattern. This is the recommended resolver when using net/http.ServeMux with
// method-and-pattern registration (e.g., "GET /users/{id}").
func StdlibPathResolver(r *http.Request) string {
	return r.Pattern
}

// RouteConfig holds HAC metadata for a specific route.
type RouteConfig struct {
	Description string
	Actions     []Action
	Related     []RelatedResource
}

// routeKey identifies a route by method and pattern.
type routeKey struct {
	method  string
	pattern string
}

// Registry stores HAC metadata for routes. It is safe for concurrent reads
// after initial configuration; concurrent writes are protected by a mutex.
type Registry struct {
	mu     sync.RWMutex
	routes map[routeKey]*RouteConfig
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		routes: make(map[routeKey]*RouteConfig),
	}
}

// Lookup returns the RouteConfig for the given method and pattern, or nil.
func (reg *Registry) Lookup(method, pattern string) *RouteConfig {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	return reg.routes[routeKey{method: method, pattern: pattern}]
}

// Routes returns all registered route keys as (method, pattern) pairs.
func (reg *Registry) Routes() [][2]string {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	pairs := make([][2]string, 0, len(reg.routes))
	for k := range reg.routes {
		pairs = append(pairs, [2]string{k.method, k.pattern})
	}
	return pairs
}

// Get starts building a route config for a GET route.
func (reg *Registry) Get(pattern string) *RouteBuilder {
	return &RouteBuilder{registry: reg, method: "GET", pattern: pattern}
}

// Post starts building a route config for a POST route.
func (reg *Registry) Post(pattern string) *RouteBuilder {
	return &RouteBuilder{registry: reg, method: "POST", pattern: pattern}
}

// Put starts building a route config for a PUT route.
func (reg *Registry) Put(pattern string) *RouteBuilder {
	return &RouteBuilder{registry: reg, method: "PUT", pattern: pattern}
}

// Patch starts building a route config for a PATCH route.
func (reg *Registry) Patch(pattern string) *RouteBuilder {
	return &RouteBuilder{registry: reg, method: "PATCH", pattern: pattern}
}

// Delete starts building a route config for a DELETE route.
func (reg *Registry) Delete(pattern string) *RouteBuilder {
	return &RouteBuilder{registry: reg, method: "DELETE", pattern: pattern}
}

// Route starts building a route config for an arbitrary method.
func (reg *Registry) Route(method, pattern string) *RouteBuilder {
	return &RouteBuilder{registry: reg, method: method, pattern: pattern}
}

// RouteBuilder provides a fluent API for configuring HAC metadata on a route.
type RouteBuilder struct {
	registry    *Registry
	method      string
	pattern     string
	description string
	actions     []Action
	related     []RelatedResource
}

// Description sets the resource description.
func (b *RouteBuilder) Description(desc string) *RouteBuilder {
	b.description = desc
	return b
}

// Actions sets the available actions for this route.
func (b *RouteBuilder) Actions(actions ...Action) *RouteBuilder {
	b.actions = actions
	return b
}

// Related sets the related resources for this route.
func (b *RouteBuilder) Related(related ...RelatedResource) *RouteBuilder {
	b.related = related
	return b
}

// Register stores the built route config in the registry.
func (b *RouteBuilder) Register() {
	cfg := &RouteConfig{
		Description: b.description,
		Actions:     b.actions,
		Related:     b.related,
	}
	b.registry.mu.Lock()
	defer b.registry.mu.Unlock()
	b.registry.routes[routeKey{method: b.method, pattern: b.pattern}] = cfg
}
