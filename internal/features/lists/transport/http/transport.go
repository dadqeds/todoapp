package lists_transport_http

import (
	"context"
	"net/http"
	"strings"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_http_server "github.com/dadqeds/todoapp/internal/core/transport/http/server"
)

type ListsHTTPHandler struct {
	listsService ListsService
	botUsername  string
}

type ListsService interface {
	GetLists(ctx context.Context, userID *int) ([]domain.ListSummary, error)
	CreateList(ctx context.Context, list domain.List) (domain.List, error)
	PatchList(ctx context.Context, id int, patch domain.ListPatch) (domain.List, error)
	DeleteList(ctx context.Context, id int) error

	CreateInvite(ctx context.Context, listID int) (domain.List, error)
	RevokeInvite(ctx context.Context, listID int) error
	JoinList(ctx context.Context, code string) (domain.List, error)
	RemoveMember(ctx context.Context, listID int, userID int) error
}

// botUsername нужен для ссылок-приглашений; пустой — ссылка не формируется.
func NewListsHTTPHandler(listsService ListsService, botUsername string) *ListsHTTPHandler {
	return &ListsHTTPHandler{
		listsService: listsService,
		botUsername:  strings.TrimPrefix(botUsername, "@"),
	}
}

func (h *ListsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{Method: http.MethodGet, Path: "/lists", Handler: h.GetLists},
		{Method: http.MethodPost, Path: "/lists", Handler: h.CreateList},
		{Method: http.MethodPatch, Path: "/lists/{id}", Handler: h.PatchList},
		{Method: http.MethodDelete, Path: "/lists/{id}", Handler: h.DeleteList},
		{Method: http.MethodPost, Path: "/lists/join", Handler: h.JoinList},
		{Method: http.MethodPost, Path: "/lists/{id}/invite", Handler: h.CreateInvite},
		{Method: http.MethodDelete, Path: "/lists/{id}/invite", Handler: h.RevokeInvite},
		{Method: http.MethodDelete, Path: "/lists/{id}/members/{user_id}", Handler: h.RemoveMember},
	}
}
