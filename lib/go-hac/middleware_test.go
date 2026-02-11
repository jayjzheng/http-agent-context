package hac

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewarePassthrough(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":1}`))
	})

	mw := Middleware(Options{})(handler)
	req := httptest.NewRequest("GET", "/users/1", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != `{"id":1}` {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestMiddlewareHACEnvelope(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":1,"name":"Alice"}`))
	})

	reg := NewRegistry()
	reg.Route("GET", "/users/1").
		Description("A user account.").
		Actions(Action{Rel: "delete", Method: "DELETE", Href: "/users/1"}).
		Register()

	mw := Middleware(Options{Registry: reg})(handler)

	req := httptest.NewRequest("GET", "/users/1", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != MediaType {
		t.Errorf("Content-Type = %q, want %q", rec.Header().Get("Content-Type"), MediaType)
	}
	if rec.Header().Get("Vary") != "Accept" {
		t.Errorf("Vary = %q, want Accept", rec.Header().Get("Vary"))
	}

	var env SuccessEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.HAC.Version != SpecVersion {
		t.Errorf("version = %q", env.HAC.Version)
	}
	if env.HAC.Description != "A user account." {
		t.Errorf("description = %q", env.HAC.Description)
	}
	if len(env.HAC.Actions) != 1 {
		t.Errorf("actions count = %d", len(env.HAC.Actions))
	}

	var data map[string]any
	json.Unmarshal(env.Data, &data)
	if data["name"] != "Alice" {
		t.Errorf("data.name = %v", data["name"])
	}
}

func TestMiddlewareErrorEnvelope(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		w.Write([]byte(`{"code":"not_found","message":"User not found"}`))
	})

	reg := NewRegistry()
	reg.Route("GET", "/users/1").Description("A user.").Register()

	mw := Middleware(Options{Registry: reg})(handler)

	req := httptest.NewRequest("GET", "/users/1", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}

	var env ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "not_found" {
		t.Errorf("error code = %q", env.Error.Code)
	}
}

func TestMiddleware406WhenHACOnly(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Empty registry — no HAC config for any route
	mw := Middleware(Options{})(handler)

	req := httptest.NewRequest("GET", "/unknown", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != 406 {
		t.Errorf("status = %d, want 406", rec.Code)
	}
}

func TestMiddlewarePassthroughWhenNoConfigButFallback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mw := Middleware(Options{})(handler)

	req := httptest.NewRequest("GET", "/unknown", nil)
	req.Header.Set("Accept", "application/vnd.hac+json, application/json;q=0.9")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want 'ok'", rec.Body.String())
	}
}

func TestMiddlewareIsHACRequested(t *testing.T) {
	var hacRequested bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hacRequested = IsHACRequested(r)
		w.Write([]byte(`{}`))
	})

	reg := NewRegistry()
	reg.Route("GET", "/test").Description("Test.").Register()
	mw := Middleware(Options{Registry: reg})(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !hacRequested {
		t.Error("IsHACRequested should be true inside handler")
	}
}

func TestMiddlewarePreservesOriginalHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.Write([]byte(`{}`))
	})

	reg := NewRegistry()
	reg.Route("GET", "/test").Description("Test.").Register()
	mw := Middleware(Options{Registry: reg})(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Header().Get("X-Custom") != "value" {
		t.Errorf("X-Custom = %q, want 'value'", rec.Header().Get("X-Custom"))
	}
}

func TestMiddlewareWithStdlibPathResolver(t *testing.T) {
	reg := NewRegistry()
	reg.Route("GET", "GET /users/{id}").
		Description("A user.").
		Actions(Action{Rel: "self", Method: "GET", Href: "/users/{id}"}).
		Register()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":1}`))
	})

	mw := Middleware(Options{
		Registry: reg,
		PathResolver: func(r *http.Request) string {
			// Simulate Go 1.22 r.Pattern
			return "GET /users/{id}"
		},
	})(handler)

	req := httptest.NewRequest("GET", "/users/1", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != MediaType {
		t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestMiddlewareCustomErrorMapper(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"detail":"has subscriptions"}`))
	})

	reg := NewRegistry()
	reg.Route("DELETE", "/users/1").Description("Delete user.").Register()

	mapper := func(statusCode int, body []byte, r *http.Request) *HACError {
		return &HACError{
			Code:    "active_subscriptions",
			Message: "Cannot delete user with active subscriptions.",
			Recovery: &Recovery{
				Description: "Cancel subscriptions first.",
				Actions: []Action{
					{Rel: "cancel-subscriptions", Method: "POST", Href: "/users/1/cancel-subscriptions"},
				},
			},
		}
	}

	mw := Middleware(Options{Registry: reg, ErrorMapper: mapper})(handler)

	req := httptest.NewRequest("DELETE", "/users/1", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	var env ErrorEnvelope
	json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "active_subscriptions" {
		t.Errorf("code = %q", env.Error.Code)
	}
	if env.Error.Recovery == nil || len(env.Error.Recovery.Actions) != 1 {
		t.Error("expected recovery with 1 action")
	}
}

// Ensure no body is leaked to the variable to keep the linter happy
var _ io.Writer = httptest.NewRecorder()
