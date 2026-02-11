package hac

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoveryHandler(t *testing.T) {
	disc := &Discovery{
		Meta: &DiscoveryMeta{
			Name:        "Test API",
			Version:     "1.0",
			Description: "A test API.",
			Resources: []ResourceEntry{
				{Rel: "users", Href: "/users", Methods: []string{"GET", "POST"}},
			},
		},
	}

	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	handler := disc.Handler(fallback)

	t.Run("HAC request returns discovery", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept", "application/vnd.hac+json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Header().Get("Content-Type") != MediaType {
			t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
		}

		var resp DiscoveryResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.HAC.Name != "Test API" {
			t.Errorf("name = %q", resp.HAC.Name)
		}
		if len(resp.HAC.Resources) != 1 {
			t.Errorf("resources count = %d", len(resp.HAC.Resources))
		}
	})

	t.Run("non-HAC request delegates to fallback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Body.String() != `{"status":"ok"}` {
			t.Errorf("body = %q", rec.Body.String())
		}
	})
}

func TestDiscoveryHandlerNilFallback(t *testing.T) {
	disc := &Discovery{Meta: &DiscoveryMeta{Name: "API", Resources: []ResourceEntry{}}}
	handler := disc.Handler(nil)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestAutoDiscovery(t *testing.T) {
	reg := NewRegistry()
	reg.Get("/users").Description("List users.").Register()
	reg.Post("/users").Description("Create user.").Register()
	reg.Get("/users/{id}").Description("Get user.").Register()
	reg.Delete("/users/{id}").Description("Delete user.").Register()
	reg.Get("/orders").Description("List orders.").Register()

	disc := AutoDiscovery("My API", "2.0", "A sample API.", reg)

	if disc.Meta.Name != "My API" {
		t.Errorf("name = %q", disc.Meta.Name)
	}
	if disc.Meta.Version != "2.0" {
		t.Errorf("version = %q", disc.Meta.Version)
	}

	// Should have grouped: /users, /users/{id}, /orders = 3 resources
	if len(disc.Meta.Resources) != 3 {
		t.Errorf("resources count = %d, want 3", len(disc.Meta.Resources))
		for _, r := range disc.Meta.Resources {
			t.Logf("  %s %s %v", r.Rel, r.Href, r.Methods)
		}
	}

	// Check sorted order
	if len(disc.Meta.Resources) >= 2 {
		if disc.Meta.Resources[0].Href > disc.Meta.Resources[1].Href {
			t.Error("resources should be sorted by href")
		}
	}
}

func TestAutoDiscoveryWithStdlibPatterns(t *testing.T) {
	reg := NewRegistry()
	reg.Route("GET", "GET /items").Description("List items.").Register()
	reg.Route("POST", "POST /items").Description("Create item.").Register()

	disc := AutoDiscovery("Item API", "1.0", "", reg)

	if len(disc.Meta.Resources) != 1 {
		t.Fatalf("resources count = %d, want 1 (should be grouped)", len(disc.Meta.Resources))
	}
	if disc.Meta.Resources[0].Href != "/items" {
		t.Errorf("href = %q, want /items", disc.Meta.Resources[0].Href)
	}
	if len(disc.Meta.Resources[0].Methods) != 2 {
		t.Errorf("methods count = %d, want 2", len(disc.Meta.Resources[0].Methods))
	}
}

func TestDeriveRel(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"/users", "users"},
		{"/users/{id}", "users"},
		{"/orders/{id}/items", "orders"},
		{"", "root"},
	}
	for _, tt := range tests {
		got := deriveRel(tt.pattern)
		if got != tt.want {
			t.Errorf("deriveRel(%q) = %q, want %q", tt.pattern, got, tt.want)
		}
	}
}
