package lists_transport_http

import (
	"context"
	"net/http"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_http_server "github.com/dadqeds/todoapp/internal/core/transport/http/server"
)

type ListsHTTPHandler struct {
	listsService ListsService
}

type ListsService interface {
	GetLists(ctx context.Context, userID *int) ([]domain.ListSummary, error)
	CreateList(ctx context.Context, list domain.List) (domain.List, error)
	PatchList(ctx context.Context, id int, patch domain.ListPatch) (domain.List, error)
	DeleteList(ctx context.Context, id int) error
}

func NewListsHTTPHandler(listsService ListsService) *ListsHTTPHandler {
	return &ListsHTTPHandler{
		listsService: listsService,
	}
}

func (h *ListsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{Method: http.MethodGet, Path: "/lists", Handler: h.GetLists},
		{Method: http.MethodPost, Path: "/lists", Handler: h.CreateList},
		{Method: http.MethodPatch, Path: "/lists/{id}", Handler: h.PatchList},
		{Method: http.MethodDelete, Path: "/lists/{id}", Handler: h.DeleteList},
	}
}
