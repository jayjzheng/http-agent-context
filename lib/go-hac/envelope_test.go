package hac

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildSuccessEnvelope(t *testing.T) {
	cfg := &RouteConfig{
		Description: "A user.",
		Actions:     []Action{{Rel: "edit", Method: "PUT", Href: "/users/1"}},
	}
	env, err := buildSuccessEnvelope([]byte(`{"id":1}`), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.HAC.Version != SpecVersion {
		t.Errorf("version = %q, want %q", env.HAC.Version, SpecVersion)
	}
	if env.HAC.Description != "A user." {
		t.Errorf("description = %q", env.HAC.Description)
	}
	if string(env.Data) != `{"id":1}` {
		t.Errorf("data = %s", env.Data)
	}
}

func TestBuildSuccessEnvelopeEmptyBody(t *testing.T) {
	env, err := buildSuccessEnvelope(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(env.Data) != "null" {
		t.Errorf("data = %s, want null", env.Data)
	}
}

func TestBuildSuccessEnvelopeInvalidJSON(t *testing.T) {
	env, err := buildSuccessEnvelope([]byte("not json"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(env.Data) != "null" {
		t.Errorf("data = %s, want null", env.Data)
	}
}

func TestBuildErrorEnvelopeDefault(t *testing.T) {
	body := []byte(`{"code":"bad_input","message":"Invalid email"}`)
	r := httptest.NewRequest("POST", "/users", nil)
	env, err := buildErrorEnvelope(400, body, r, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Error.Code != "bad_input" {
		t.Errorf("code = %q, want bad_input", env.Error.Code)
	}
	if env.Error.Message != "Invalid email" {
		t.Errorf("message = %q", env.Error.Message)
	}
}

func TestBuildErrorEnvelopeEmptyBody(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	env, _ := buildErrorEnvelope(404, nil, r, nil)
	if env.Error.Code != "Not Found" {
		t.Errorf("code = %q, want 'Not Found'", env.Error.Code)
	}
}

func TestBuildErrorEnvelopeRetryable(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	env, _ := buildErrorEnvelope(500, nil, r, nil)
	if !env.Error.Retryable {
		t.Error("500 should be retryable by default")
	}

	env, _ = buildErrorEnvelope(429, nil, r, nil)
	if !env.Error.Retryable {
		t.Error("429 should be retryable by default")
	}

	env, _ = buildErrorEnvelope(400, nil, r, nil)
	if env.Error.Retryable {
		t.Error("400 should not be retryable")
	}
}

func TestBuildErrorEnvelopeCustomMapper(t *testing.T) {
	mapper := func(statusCode int, body []byte, r *http.Request) *HACError {
		return &HACError{
			Code:    "custom_error",
			Message: "Custom message",
			Recovery: &Recovery{
				Description: "Try again.",
			},
		}
	}
	r := httptest.NewRequest("POST", "/", nil)
	env, _ := buildErrorEnvelope(422, nil, r, mapper)
	if env.Error.Code != "custom_error" {
		t.Errorf("code = %q, want custom_error", env.Error.Code)
	}
	if env.Error.Recovery == nil {
		t.Error("expected recovery")
	}
}

func TestBuildErrorEnvelopeErrorField(t *testing.T) {
	body := []byte(`{"error":"Something went wrong"}`)
	r := httptest.NewRequest("GET", "/", nil)
	env, _ := buildErrorEnvelope(500, body, r, nil)
	if env.Error.Message != "Something went wrong" {
		t.Errorf("message = %q, want 'Something went wrong'", env.Error.Message)
	}
}

// http import needed for ErrorMapper type in test
var _ ErrorMapper = func(int, []byte, *http.Request) *HACError { return nil }

// Ensure envelopes marshal to valid JSON
func TestEnvelopeJSON(t *testing.T) {
	cfg := &RouteConfig{Description: "Test."}
	env, _ := buildSuccessEnvelope([]byte(`[1,2,3]`), cfg)
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Error("output is not valid JSON")
	}
}
