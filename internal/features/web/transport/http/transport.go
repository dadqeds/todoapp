package web_transport_http

import (
	"io/fs"
	"net/http"

	core_http_server "github.com/dadqeds/todoapp/internal/core/transport/http/server"
)

type WebHTTPHandler struct {
	webService WebService
	assets     http.HandlerFunc
}

type WebService interface {
	GetMainPage() ([]byte, error)
}

// files — корень фронтенда; статика берётся из его подкаталога assets/.
func NewWebHTTPHandler(
	webService WebService,
	files fs.FS,
) *WebHTTPHandler {
	return &WebHTTPHandler{
		webService: webService,
		assets:     newAssetsHandler(files),
	}
}

func (h *WebHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodGet,
			Path:    "/{$}",
			Handler: h.GetMainPage,
		},
		{
			Method:  http.MethodGet,
			Path:    "/assets/",
			Handler: h.assets,
		},
	}
}
