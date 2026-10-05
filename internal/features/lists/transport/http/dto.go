package lists_transport_http

import (
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

type ListDTOResponse struct {
	ID          int       `json:"id"            example:"3"`
	Version     int       `json:"version"       example:"1"`
	Title       string    `json:"title"         example:"Дом"`
	Color       string    `json:"color"         example:"green" enums:"green,violet,coral,blue,pink,amber"`
	OwnerUserID int       `json:"owner_user_id" example:"1"`
	IsDefault   bool      `json:"is_default"    example:"false"`
	CreatedAt   time.Time `json:"created_at"`
}

type ListSummaryDTOResponse struct {
	ListDTOResponse
	OpenTasks  int `json:"open_tasks"  example:"3"`
	TotalTasks int `json:"total_tasks" example:"5"`
}

func listDTOFromDomain(l domain.List) ListDTOResponse {
	return ListDTOResponse{
		ID:          l.ID,
		Version:     l.Version,
		Title:       l.Title,
		Color:       l.Color,
		OwnerUserID: l.OwnerUserID,
		IsDefault:   l.IsDefault,
		CreatedAt:   l.CreatedAt,
	}
}
