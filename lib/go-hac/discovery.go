package hac

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

// Discovery serves a HAC discovery document for HAC-requesting clients and
// delegates to a fallback handler otherwise.
type Discovery struct {
	// Meta is the discovery metadata to serve.
	Meta *DiscoveryMeta
}

// Handler returns an http.Handler that serves the discovery document for HAC
// requests and delegates to fallback for all others. If fallback is nil, non-HAC
// requests receive a 404.
func (d *Discovery) Handler(fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if !wantsHAC(accept) {
			if fallback != nil {
				fallback.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
			return
		}

		resp := &DiscoveryResponse{HAC: d.Meta}
		out, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", MediaType)
		w.Header().Set("Vary", "Accept")
		w.Write(out)
	})
}

// AutoDiscovery generates a Discovery from the routes registered in the given
// Registry. It groups routes by pattern and collects their methods.
func AutoDiscovery(name, version, description string, reg *Registry) *Discovery {
	routes := reg.Routes()

	// Group methods by pattern
	type entry struct {
		pattern string
		methods []string
	}
	grouped := make(map[string]*entry)
	for _, pair := range routes {
		method, pattern := pair[0], pair[1]
		// Strip method prefix from stdlib patterns like "GET /users/{id}"
		cleanPattern := pattern
		if idx := strings.Index(pattern, " /"); idx >= 0 {
			cleanPattern = pattern[idx+1:]
		}
		e, ok := grouped[cleanPattern]
		if !ok {
			e = &entry{pattern: cleanPattern}
			grouped[cleanPattern] = e
		}
		e.methods = append(e.methods, method)
	}

	// Build sorted resource entries
	resources := make([]ResourceEntry, 0, len(grouped))
	for _, e := range grouped {
		sort.Strings(e.methods)
		// Derive rel from pattern: "/users/{id}" -> "users"
		rel := deriveRel(e.pattern)

		// Get description from the first route config we find
		var desc string
		for _, pair := range routes {
			method, pat := pair[0], pair[1]
			cleanPat := pat
			if idx := strings.Index(pat, " /"); idx >= 0 {
				cleanPat = pat[idx+1:]
			}
			if cleanPat == e.pattern {
				if cfg := reg.Lookup(method, pat); cfg != nil && cfg.Description != "" {
					desc = cfg.Description
					break
				}
			}
		}

		resources = append(resources, ResourceEntry{
			Rel:         rel,
			Href:        e.pattern,
			Description: desc,
			Methods:     e.methods,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Href < resources[j].Href
	})

	return &Discovery{
		Meta: &DiscoveryMeta{
			Name:        name,
			Version:     version,
			Description: description,
			Resources:   resources,
		},
	}
}

// deriveRel extracts a relation name from a URL pattern.
// "/users/{id}" -> "users", "/orders" -> "orders"
func deriveRel(pattern string) string {
	pattern = strings.TrimPrefix(pattern, "/")
	parts := strings.Split(pattern, "/")
	for _, p := range parts {
		if p != "" && !strings.HasPrefix(p, "{") {
			return p
		}
	}
	if pattern == "" {
		return "root"
	}
	return pattern
}
