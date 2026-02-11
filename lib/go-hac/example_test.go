package hac_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	hac "github.com/jayjzheng/http-agent-context/lib/go-hac"
)

func Example() {
	// 1. Create a registry and register HAC metadata for routes
	reg := hac.NewRegistry()
	reg.Route("GET", "/users/1").
		Description("A user account. Contains PII — do not log response bodies.").
		Actions(
			hac.Action{
				Rel:         "delete",
				Method:      "DELETE",
				Href:        "/users/1",
				Description: "Permanently delete this user and all associated data.",
				Safety: &hac.Safety{
					Mutability:              hac.Irreversible,
					BlastRadius:             hac.SelfAndAssociated,
					ConfirmationRecommended: true,
				},
			},
		).
		Register()

	// 2. Create your normal HTTP handler
	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":1,"name":"Alice","email":"alice@example.com"}`))
	})

	// 3. Wrap with HAC middleware
	wrapped := hac.Middleware(hac.Options{Registry: reg})(userHandler)

	// 4. Test with a HAC-requesting client
	req := httptest.NewRequest("GET", "/users/1", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))
	fmt.Println("Vary:", rec.Header().Get("Vary"))
	fmt.Println("Status:", rec.Code)
	// Output:
	// Content-Type: application/vnd.hac+json
	// Vary: Accept
	// Status: 200
}

func Example_passthrough() {
	reg := hac.NewRegistry()
	reg.Route("GET", "/users/1").Description("A user.").Register()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":1}`))
	})

	wrapped := hac.Middleware(hac.Options{Registry: reg})(handler)

	// Non-HAC request passes through unchanged
	req := httptest.NewRequest("GET", "/users/1", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))
	body, _ := io.ReadAll(rec.Body)
	fmt.Println("Body:", string(body))
	// Output:
	// Content-Type: application/json
	// Body: {"id":1}
}

func Example_discovery() {
	reg := hac.NewRegistry()
	reg.Route("GET", "/users").Description("List all users.").Register()
	reg.Route("POST", "/users").Description("Create a new user.").Register()

	disc := hac.AutoDiscovery("My API", "1.0", "A sample REST API.", reg)
	handler := disc.Handler(nil)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/vnd.hac+json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println("Content-Type:", rec.Header().Get("Content-Type"))
	fmt.Println("Status:", rec.Code)
	// Output:
	// Content-Type: application/vnd.hac+json
	// Status: 200
}
