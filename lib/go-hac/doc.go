// Package hac provides HTTP middleware that implements the HTTP Agent Context
// (HAC) specification. It enriches API responses with agent-oriented metadata
// via content negotiation, allowing AI agents to discover available actions,
// assess safety, and recover from errors—without affecting non-agent clients.
//
// Agents request HAC metadata by sending Accept: application/vnd.hac+json.
// The middleware wraps the original JSON response in a HAC envelope containing
// the original data plus a _hac metadata block with actions, safety info, and
// related resources.
//
// # Quick Start
//
//	reg := hac.NewRegistry()
//	reg.Get("/users/{id}").
//		Description("A user account.").
//		Actions(hac.Action{
//			Rel: "delete", Method: "DELETE", Href: "/users/{id}",
//			Description: "Permanently delete this user.",
//			Safety: &hac.Safety{
//				Mutability:              hac.Irreversible,
//				BlastRadius:             hac.SelfAndAssociated,
//				ConfirmationRecommended: true,
//			},
//		}).
//		Register()
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("GET /users/{id}", userHandler)
//
//	wrapped := hac.Middleware(hac.Options{
//		Registry:     reg,
//		PathResolver: hac.StdlibPathResolver,
//	})(mux)
//
//	http.ListenAndServe(":8080", wrapped)
package hac
