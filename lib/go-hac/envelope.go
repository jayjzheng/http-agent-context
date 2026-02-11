package hac

import (
	"encoding/json"
	"net/http"
)

// buildSuccessEnvelope wraps the original response body in a HAC success envelope.
func buildSuccessEnvelope(body []byte, cfg *RouteConfig) (*SuccessEnvelope, error) {
	// Validate that body is valid JSON; use null if empty or invalid
	var raw json.RawMessage
	if len(body) > 0 && json.Valid(body) {
		raw = body
	} else {
		raw = json.RawMessage("null")
	}

	meta := &HACMeta{
		Version: SpecVersion,
	}
	if cfg != nil {
		meta.Description = cfg.Description
		meta.Actions = cfg.Actions
		meta.Related = cfg.Related
	}

	return &SuccessEnvelope{
		Data: raw,
		HAC:  meta,
	}, nil
}

// ErrorMapper is a callback that converts an HTTP error response into a HACError.
// It receives the status code, the original response body, and the request.
// Return nil to use the default error mapping.
type ErrorMapper func(statusCode int, body []byte, r *http.Request) *HACError

// buildErrorEnvelope constructs a HAC error envelope from the response.
func buildErrorEnvelope(statusCode int, body []byte, r *http.Request, mapper ErrorMapper) (*ErrorEnvelope, error) {
	if mapper != nil {
		if hacErr := mapper(statusCode, body, r); hacErr != nil {
			return &ErrorEnvelope{Error: hacErr}, nil
		}
	}

	hacErr := defaultErrorMapping(statusCode, body)
	return &ErrorEnvelope{Error: hacErr}, nil
}

// defaultErrorMapping tries to extract code/message from the original JSON body,
// falling back to the HTTP status text.
func defaultErrorMapping(statusCode int, body []byte) *HACError {
	hacErr := &HACError{
		Code:    http.StatusText(statusCode),
		Message: http.StatusText(statusCode),
	}

	// Mark 429 and 5xx as retryable by default
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		hacErr.Retryable = true
	}

	if len(body) == 0 {
		return hacErr
	}

	// Try to extract code and message from JSON body
	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if parsed.Code != "" {
			hacErr.Code = parsed.Code
		}
		if parsed.Message != "" {
			hacErr.Message = parsed.Message
		} else if parsed.Error != "" {
			hacErr.Message = parsed.Error
		}
	}

	return hacErr
}
