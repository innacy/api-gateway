// internal/pkg/response/json.go
package response

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, code string, message string) {
	JSON(w, status, ErrorBody{
		Error:   code,
		Message: message,
	})
}

func ErrorWithRequestID(w http.ResponseWriter, status int, code, message, requestID string) {
	JSON(w, status, ErrorBody{
		Error:     code,
		Message:   message,
		RequestID: requestID,
	})
}
