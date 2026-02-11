package hac

import (
	"context"
	"net/http"
)

type contextKey struct{}

// IsHACRequested reports whether the given request was identified as a HAC
// request by the middleware (i.e., the Accept header included the HAC media type).
// This can be used by handlers to customize behavior for agent clients.
func IsHACRequested(r *http.Request) bool {
	v, _ := r.Context().Value(contextKey{}).(bool)
	return v
}

// withHACRequested returns a new context with the HAC-requested flag set.
func withHACRequested(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, true)
}
