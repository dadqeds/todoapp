package core_http_middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestCORS(t *testing.T) {
	h := CORS([]string{"http://allowed.example"})(okHandler)

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://allowed.example")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://allowed.example" {
			t.Fatalf("Allow-Origin = %q", got)
		}
		if rec.Header().Get("Vary") != "Origin" {
			t.Fatalf("Vary = %q", rec.Header().Get("Vary"))
		}
	})

	for _, origin := range []string{"http://evil.example", "null"} {
		t.Run("rejected origin "+origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("Allow-Origin = %q, want empty", got)
			}
		})
	}

	t.Run("preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://allowed.example")
		req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("code = %d", rec.Code)
		}
	})
}

func TestRequestID(t *testing.T) {
	var seen string
	h := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get(RequestIDHeader)
	}))

	tests := []struct {
		name     string
		incoming string
		keep     bool
	}{
		{"empty is generated", "", false},
		{"valid is kept", "abc-123_x.y", true},
		{"too long is replaced", strings.Repeat("a", 65), false},
		{"unsafe chars are replaced", "id\nInjected: 1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.incoming != "" {
				req.Header.Set(RequestIDHeader, tt.incoming)
			}
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if tt.keep && seen != tt.incoming {
				t.Fatalf("request id = %q, want %q", seen, tt.incoming)
			}
			if !tt.keep && (seen == tt.incoming || !requestIDRegexp.MatchString(seen)) {
				t.Fatalf("request id = %q was not regenerated", seen)
			}
			if rec.Header().Get(RequestIDHeader) != seen {
				t.Fatalf("response header = %q, want %q", rec.Header().Get(RequestIDHeader), seen)
			}
		})
	}
}
