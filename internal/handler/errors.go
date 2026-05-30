package handler

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, message string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIError{Error: message, Details: details})
}
