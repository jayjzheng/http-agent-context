package hac

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// Options configures the HAC middleware.
type Options struct {
	// Registry contains HAC metadata for routes.
	Registry *Registry

	// PathResolver extracts the route pattern from a request.
	// Defaults to using r.URL.Path if nil.
	PathResolver PathResolver

	// ErrorMapper optionally customizes error-to-HACError conversion.
	ErrorMapper ErrorMapper
}

// Middleware returns an http.Handler middleware that wraps responses in HAC
// envelopes when the client sends Accept: application/vnd.hac+json.
func Middleware(opts Options) func(http.Handler) http.Handler {
	if opts.PathResolver == nil {
		opts.PathResolver = func(r *http.Request) string {
			return r.URL.Path
		}
	}
	if opts.Registry == nil {
		opts.Registry = NewRegistry()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accept := r.Header.Get("Accept")

			if !wantsHAC(accept) {
				next.ServeHTTP(w, r)
				return
			}

			// Resolve route and look up config
			pattern := opts.PathResolver(r)
			cfg := opts.Registry.Lookup(r.Method, pattern)

			if cfg == nil {
				// No HAC config for this route
				if hacIsOnlyAcceptable(accept) {
					// Client only accepts HAC, but we can't provide it
					http.Error(w, "Not Acceptable: no HAC metadata for this route", http.StatusNotAcceptable)
					return
				}
				// Other types acceptable, passthrough
				next.ServeHTTP(w, r)
				return
			}

			// Set HAC-requested flag in context
			r = r.WithContext(withHACRequested(r.Context()))

			// Capture the response
			rec := &responseRecorder{
				header: make(http.Header),
				body:   &bytes.Buffer{},
				code:   http.StatusOK,
			}
			next.ServeHTTP(rec, r)

			// Build envelope
			var envelope any
			var err error
			if rec.code >= 400 {
				envelope, err = buildErrorEnvelope(rec.code, rec.body.Bytes(), r, opts.ErrorMapper)
			} else {
				envelope, err = buildSuccessEnvelope(rec.body.Bytes(), cfg)
			}

			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			out, err := json.Marshal(envelope)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Copy headers from recorded response, then override content type
			for k, vs := range rec.header {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("Content-Type", MediaType)
			w.Header().Set("Vary", "Accept")

			if rec.code >= 400 {
				w.WriteHeader(rec.code)
			}
			w.Write(out)
		})
	}
}

// responseRecorder captures the status code, headers, and body written by a handler.
type responseRecorder struct {
	header http.Header
	body   *bytes.Buffer
	code   int
	wrote  bool
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.wrote = true
	}
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.code = statusCode
}
