package hac

import "testing"

func TestRegistryLookup(t *testing.T) {
	reg := NewRegistry()
	reg.Get("/users/{id}").
		Description("A user account.").
		Actions(Action{Rel: "delete", Method: "DELETE", Href: "/users/{id}"}).
		Register()

	cfg := reg.Lookup("GET", "/users/{id}")
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Description != "A user account." {
		t.Errorf("description = %q", cfg.Description)
	}
	if len(cfg.Actions) != 1 {
		t.Errorf("actions count = %d, want 1", len(cfg.Actions))
	}
}

func TestRegistryLookupMiss(t *testing.T) {
	reg := NewRegistry()
	if cfg := reg.Lookup("GET", "/nope"); cfg != nil {
		t.Error("expected nil for unregistered route")
	}
}

func TestRegistryRoutes(t *testing.T) {
	reg := NewRegistry()
	reg.Get("/users").Description("Users list.").Register()
	reg.Post("/users").Description("Create user.").Register()
	reg.Delete("/users/{id}").Description("Delete user.").Register()

	routes := reg.Routes()
	if len(routes) != 3 {
		t.Errorf("routes count = %d, want 3", len(routes))
	}
}

func TestRouteBuilderFluent(t *testing.T) {
	reg := NewRegistry()
	reg.Put("/orders/{id}").
		Description("An order.").
		Actions(
			Action{Rel: "cancel", Method: "POST", Href: "/orders/{id}/cancel"},
		).
		Related(
			RelatedResource{Rel: "items", Href: "/orders/{id}/items"},
		).
		Register()

	cfg := reg.Lookup("PUT", "/orders/{id}")
	if cfg == nil {
		t.Fatal("expected config")
	}
	if len(cfg.Related) != 1 {
		t.Errorf("related count = %d, want 1", len(cfg.Related))
	}
}

func TestRouteMethod(t *testing.T) {
	reg := NewRegistry()
	reg.Route("OPTIONS", "/test").Description("Options test.").Register()

	if cfg := reg.Lookup("OPTIONS", "/test"); cfg == nil {
		t.Error("expected config for OPTIONS route")
	}
}

func TestPatchBuilder(t *testing.T) {
	reg := NewRegistry()
	reg.Patch("/items/{id}").Description("Patch item.").Register()
	if cfg := reg.Lookup("PATCH", "/items/{id}"); cfg == nil {
		t.Error("expected config for PATCH route")
	}
}
