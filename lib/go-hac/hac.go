package hac

import "encoding/json"

// HAC media type and spec version constants.
const (
	MediaType   = "application/vnd.hac+json"
	SpecVersion = "1.0"
)

// Mutability indicates whether an action mutates state and if the mutation is reversible.
type Mutability string

const (
	ReadOnly    Mutability = "read_only"
	Reversible  Mutability = "reversible"
	Irreversible Mutability = "irreversible"
)

// BlastRadius indicates the scope of resources affected by an action.
type BlastRadius string

const (
	Self              BlastRadius = "self"
	SelfAndAssociated BlastRadius = "self_and_associated"
	Many              BlastRadius = "many"
	All               BlastRadius = "all"
)

// SuccessEnvelope is the HAC success response wrapping original data with metadata.
type SuccessEnvelope struct {
	Data json.RawMessage `json:"data"`
	HAC  *HACMeta        `json:"_hac"`
}

// HACMeta contains agent-oriented metadata about a resource.
type HACMeta struct {
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Actions     []Action          `json:"actions,omitempty"`
	Related     []RelatedResource `json:"related,omitempty"`
}

// Action represents a hypermedia action an agent can invoke.
type Action struct {
	Rel           string   `json:"rel"`
	Method        string   `json:"method"`
	Href          string   `json:"href"`
	Description   string   `json:"description,omitempty"`
	Safety        *Safety  `json:"safety,omitempty"`
	Fields        []Field  `json:"fields,omitempty"`
	Preconditions []string `json:"preconditions,omitempty"`
}

// Safety contains risk-assessment metadata for an action.
type Safety struct {
	Mutability              Mutability  `json:"mutability,omitempty"`
	BlastRadius             BlastRadius `json:"blast_radius,omitempty"`
	ReversibleWithin        string      `json:"reversible_within,omitempty"`
	ConfirmationRecommended bool        `json:"confirmation_recommended,omitempty"`
	Cost                    *Cost       `json:"cost,omitempty"`
}

// Cost represents the financial cost of performing an action.
type Cost struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description,omitempty"`
}

// Field represents an input field for an action.
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
	Default     any    `json:"default,omitempty"`
}

// RelatedResource is a link to a related resource.
type RelatedResource struct {
	Rel         string `json:"rel"`
	Href        string `json:"href"`
	Description string `json:"description,omitempty"`
}

// ErrorEnvelope is the HAC error response envelope.
type ErrorEnvelope struct {
	Error *HACError `json:"error"`
}

// HACError contains structured error info with optional recovery guidance.
type HACError struct {
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	Retryable  bool      `json:"retryable,omitempty"`
	RetryAfter int       `json:"retry_after,omitempty"`
	Recovery   *Recovery `json:"recovery,omitempty"`
}

// Recovery provides guidance on how to resolve an error.
type Recovery struct {
	Description string   `json:"description"`
	Actions     []Action `json:"actions,omitempty"`
}

// DiscoveryResponse is the HAC discovery document returned from the API root.
type DiscoveryResponse struct {
	HAC *DiscoveryMeta `json:"_hac"`
}

// DiscoveryMeta describes the API and its available resources.
type DiscoveryMeta struct {
	Name        string          `json:"name"`
	Version     string          `json:"version,omitempty"`
	Description string          `json:"description,omitempty"`
	Resources   []ResourceEntry `json:"resources"`
}

// ResourceEntry is a discoverable API resource.
type ResourceEntry struct {
	Rel         string   `json:"rel"`
	Href        string   `json:"href"`
	Description string   `json:"description,omitempty"`
	Methods     []string `json:"methods,omitempty"`
}
