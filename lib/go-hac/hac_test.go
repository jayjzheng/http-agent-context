package hac

import (
	"encoding/json"
	"testing"
)

func TestSuccessEnvelopeMarshal(t *testing.T) {
	env := SuccessEnvelope{
		Data: json.RawMessage(`{"id":1,"name":"Alice"}`),
		HAC: &HACMeta{
			Version:     "1.0",
			Description: "A user account.",
			Actions: []Action{
				{Rel: "delete", Method: "DELETE", Href: "/users/1"},
			},
		},
	}
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := parsed["data"]; !ok {
		t.Error("missing 'data' key")
	}
	if _, ok := parsed["_hac"]; !ok {
		t.Error("missing '_hac' key")
	}
}

func TestErrorEnvelopeMarshal(t *testing.T) {
	env := ErrorEnvelope{
		Error: &HACError{
			Code:      "not_found",
			Message:   "User not found",
			Retryable: false,
			Recovery: &Recovery{
				Description: "Check the user ID and try again.",
			},
		},
	}
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	errObj, ok := parsed["error"].(map[string]any)
	if !ok {
		t.Fatal("missing 'error' object")
	}
	if errObj["code"] != "not_found" {
		t.Errorf("code = %v, want not_found", errObj["code"])
	}
}

func TestSafetyOmitEmpty(t *testing.T) {
	a := Action{
		Rel:    "view",
		Method: "GET",
		Href:   "/users/1",
	}
	out, _ := json.Marshal(a)
	var parsed map[string]any
	json.Unmarshal(out, &parsed)
	if _, ok := parsed["safety"]; ok {
		t.Error("safety should be omitted when nil")
	}
}

func TestEnumConstants(t *testing.T) {
	if ReadOnly != "read_only" {
		t.Errorf("ReadOnly = %q", ReadOnly)
	}
	if Irreversible != "irreversible" {
		t.Errorf("Irreversible = %q", Irreversible)
	}
	if SelfAndAssociated != "self_and_associated" {
		t.Errorf("SelfAndAssociated = %q", SelfAndAssociated)
	}
	if All != "all" {
		t.Errorf("All = %q", All)
	}
}
