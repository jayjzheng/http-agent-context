# go-hac

Go middleware for the [HTTP Agent Context (HAC)](../../spec/hac-spec-v1.md) specification. Drop it into any `net/http`-compatible server to serve agent-oriented metadata via content negotiation — zero external dependencies.

Agents request HAC by sending `Accept: application/vnd.hac+json`. Non-agent clients are completely unaffected.

## Install

```bash
go get github.com/jayjzheng/http-agent-context/lib/go-hac
```

Requires **Go 1.23+**.

## Quick start

```go
package main

import (
	"encoding/json"
	"net/http"

	hac "github.com/jayjzheng/http-agent-context/lib/go-hac"
)

func main() {
	// 1. Register HAC metadata for your routes
	reg := hac.NewRegistry()

	reg.Get("/users/{id}").
		Description("A user account. Contains PII — do not log response bodies.").
		Actions(
			hac.Action{
				Rel:         "delete",
				Method:      "DELETE",
				Href:        "/users/{id}",
				Description: "Permanently delete this user and all associated data.",
				Safety: &hac.Safety{
					Mutability:              hac.Irreversible,
					BlastRadius:             hac.SelfAndAssociated,
					ConfirmationRecommended: true,
				},
			},
		).
		Register()

	// 2. Set up your handlers as usual
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "name": "Alice", "email": "alice@example.com",
		})
	})

	// 3. Wrap with HAC middleware
	wrapped := hac.Middleware(hac.Options{
		Registry:     reg,
		PathResolver: hac.StdlibPathResolver,
	})(mux)

	http.ListenAndServe(":8080", wrapped)
}
```

### Normal client — unchanged

```bash
curl localhost:8080/users/1
```
```json
{"id":1,"name":"Alice","email":"alice@example.com"}
```

### Agent client — HAC envelope

```bash
curl -H "Accept: application/vnd.hac+json" localhost:8080/users/1
```
```json
{
  "data": {"id":1,"name":"Alice","email":"alice@example.com"},
  "_hac": {
    "version": "1.0",
    "description": "A user account. Contains PII — do not log response bodies.",
    "actions": [
      {
        "rel": "delete",
        "method": "DELETE",
        "href": "/users/{id}",
        "description": "Permanently delete this user and all associated data.",
        "safety": {
          "mutability": "irreversible",
          "blast_radius": "self_and_associated",
          "confirmation_recommended": true
        }
      }
    ]
  }
}
```

## API overview

### Registry and route builders

Register HAC metadata per route using a fluent builder:

```go
reg := hac.NewRegistry()

reg.Get("/orders").Description("List orders.").Register()
reg.Post("/orders").Description("Create an order.").Register()
reg.Delete("/orders/{id}").
	Description("An order.").
	Actions(hac.Action{...}).
	Related(hac.RelatedResource{Rel: "items", Href: "/orders/{id}/items"}).
	Register()

// Arbitrary method
reg.Route("OPTIONS", "/health").Description("Health check.").Register()
```

Available builders: `Get`, `Post`, `Put`, `Patch`, `Delete`, `Route`.

### Middleware

Standard `func(http.Handler) http.Handler` signature:

```go
hac.Middleware(hac.Options{
	Registry:     reg,
	PathResolver: hac.StdlibPathResolver, // uses Go 1.23+ r.Pattern
	ErrorMapper:  nil,                     // optional custom error mapping
})
```

**Behavior:**
- No `Accept: application/vnd.hac+json` header — passthrough, handler runs normally
- HAC requested + route registered — response wrapped in success or error envelope
- HAC requested + route not registered + no fallback types — returns 406
- HAC requested + route not registered + other types accepted — passthrough
- Sets `Content-Type: application/vnd.hac+json` and `Vary: Accept` on HAC responses

### Error handling

Errors (status >= 400) are automatically wrapped in a HAC error envelope. The middleware tries to extract `code` and `message` from the original JSON body and marks 429/5xx responses as retryable.

For custom error mapping with recovery guidance:

```go
hac.Middleware(hac.Options{
	Registry: reg,
	ErrorMapper: func(status int, body []byte, r *http.Request) *hac.HACError {
		if status == 422 {
			return &hac.HACError{
				Code:    "active_subscriptions",
				Message: "Cannot delete user with active subscriptions.",
				Recovery: &hac.Recovery{
					Description: "Cancel all subscriptions first, then retry.",
					Actions: []hac.Action{
						{Rel: "cancel-subscriptions", Method: "POST", Href: "/users/{id}/cancel-subscriptions"},
					},
				},
			}
		}
		return nil // fall back to default mapping
	},
})
```

### Discovery

Serve a HAC discovery document at your API root:

```go
disc := hac.AutoDiscovery("My API", "2.0", "A REST API for managing users and orders.", reg)
mux.Handle("/", disc.Handler(fallbackHandler))
```

`AutoDiscovery` generates the discovery document from registered routes, grouping by pattern and collecting methods. Non-HAC requests to `/` are delegated to the fallback handler.

### Context helper

Check if the current request is from a HAC-aware agent inside your handlers:

```go
func handler(w http.ResponseWriter, r *http.Request) {
	if hac.IsHACRequested(r) {
		// agent client
	}
}
```

## Types

All types map 1:1 to the [HAC JSON schemas](../../spec/schema/):

| Go type | Schema |
|---------|--------|
| `SuccessEnvelope` | `hac-envelope.schema.json` |
| `ErrorEnvelope` | `hac-error.schema.json` |
| `DiscoveryResponse` | `hac-discovery.schema.json` |
| `Action`, `Safety`, `Cost`, `Field` | `hac-envelope.schema.json#/$defs/*` |
| `Recovery` | `hac-error.schema.json#/$defs/Recovery` |

Enum constants: `ReadOnly`, `Reversible`, `Irreversible` (mutability) and `Self`, `SelfAndAssociated`, `Many`, `All` (blast radius).

## Running tests

```bash
cd lib/go-hac
go test ./...
```

## License

[CC BY 4.0](../../LICENSE) — same as the HAC specification.
