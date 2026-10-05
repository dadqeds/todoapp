package core_http_response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
)

func newHandler(rec *httptest.ResponseRecorder) *HTTPResponseHandler {
	return NewHTTPResponseHandler(core_logger.FromContext(context.Background()), rec)
}

func TestJSONResponseSetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	newHandler(rec).JSONResponse(map[string]int{"a": 1}, http.StatusCreated)

	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestHTMLResponseSetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	newHandler(rec).HTMLResponse([]byte("<html></html>"))

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantCode     int
		wantErrorTxt string
	}{
		{"invalid argument", fmt.Errorf("bad title: %w", core_errors.ErrInvalidArgument), http.StatusBadRequest, "bad title: invalid argument"},
		{"not found", fmt.Errorf("task 1: %w", core_errors.ErrNotFound), http.StatusNotFound, "task 1: not found"},
		{"conflict", fmt.Errorf("version: %w", core_errors.ErrConflict), http.StatusConflict, "version: conflict"},
		{"internal error is hidden", errors.New(`ERROR: relation "todoapp.tasks" does not exist`), http.StatusInternalServerError, "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			newHandler(rec).ErrorResponse(tt.err, "msg")

			if rec.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d", rec.Code, tt.wantCode)
			}

			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tt.wantErrorTxt || body.Message != "msg" {
				t.Fatalf("body = %+v", body)
			}
		})
	}
}

func TestResponseWriterKeepsFirstStatus(t *testing.T) {
	rw := NewResponseWriter(httptest.NewRecorder())

	rw.WriteHeader(http.StatusBadRequest)
	rw.WriteHeader(http.StatusOK)

	if rw.GetStatusCode() != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rw.GetStatusCode(), http.StatusBadRequest)
	}
}
