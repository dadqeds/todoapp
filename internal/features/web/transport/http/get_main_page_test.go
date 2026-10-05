package web_transport_http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	web_fs_repository "github.com/dadqeds/todoapp/internal/features/web/repository/file_system"
	web_service "github.com/dadqeds/todoapp/internal/features/web/service"
)

func TestMainPageRoute(t *testing.T) {
	files := fstest.MapFS{"index.html": {Data: []byte("<html>todo</html>")}}
	h := NewWebHTTPHandler(web_service.NewWebService(web_fs_repository.NewWebRepository(files)))

	mux := http.NewServeMux()
	for _, route := range h.Routes() {
		mux.Handle(route.Method+" "+route.Path, route.WithMiddleware())
	}

	t.Run("root serves index.html", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "todo") {
			t.Fatalf("code = %d, body = %q", rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("Content-Type = %q", ct)
		}
	})

	t.Run("unknown path is 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("code = %d, want 404", rec.Code)
		}
	})
}
