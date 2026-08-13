// internal/pkg/response/json_test.go
package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/widasinnacy/api-gateway/internal/pkg/response"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"result": "hello"},
			wantStatus: http.StatusOK,
			wantBody:   `{"result":"hello"}`,
		},
		{
			name:       "nil data returns null",
			status:     http.StatusNoContent,
			data:       nil,
			wantStatus: http.StatusNoContent,
			wantBody:   `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			response.JSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			got := w.Body.String()
			// Normalize for comparison
			var gotJSON, wantJSON any
			json.Unmarshal([]byte(got), &gotJSON)
			json.Unmarshal([]byte(tt.wantBody), &wantJSON)
			gotBytes, _ := json.Marshal(gotJSON)
			wantBytes, _ := json.Marshal(wantJSON)
			if string(gotBytes) != string(wantBytes) {
				t.Errorf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}

	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	json.NewDecoder(w.Body).Decode(&body)

	if body.Error != "unauthorized" {
		t.Errorf("error = %q, want unauthorized", body.Error)
	}
	if body.Message != "invalid token" {
		t.Errorf("message = %q, want 'invalid token'", body.Message)
	}
}
