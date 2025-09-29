package httpx

import (
	"encoding/json"
	"net/http"
)

// ResponseStatusType identifies the type of response being sent.
type ResponseStatusType string

const (
	// Error indicates an error response.
	Error ResponseStatusType = "error"
	// Success indicates a successful response.
	Success ResponseStatusType = "success"
)

// ResponseStatus is the standard status envelope returned by the helpers.
type ResponseStatus struct {
	Type        ResponseStatusType `json:"type"`
	Code        string             `json:"code"`
	Description string             `json:"description,omitempty"`
}

// Response represents a generic JSON response with an optional data payload.
type Response struct {
	Data   any            `json:"data,omitempty"`
	Status ResponseStatus `json:"status"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// FinalOutput writes the provided payload as JSON with the given status code.
func FinalOutput(w http.ResponseWriter, status int, payload any) {
	writeJSON(w, status, payload)
}

// ErrorOutput writes the provided payload as JSON, adding defensive headers
// suitable for error responses.
func ErrorOutput(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	writeJSON(w, status, payload)
}
