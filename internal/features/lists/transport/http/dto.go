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
	// Только для владельца: код и ссылка приглашения, null — приглашение выключено.
	InviteCode *string `json:"invite_code,omitempty" example:"pQ3x_Zr8k1LmN0aB"`
	InviteLink *string `json:"invite_link,omitempty" example:"https://t.me/my_todo_bot?startapp=join_pQ3x_Zr8k1LmN0aB"`
}

type ListMemberDTO struct {
	UserID   int       `json:"user_id"   example:"7"`
	FullName string    `json:"full_name" example:"Анна"`
	Role     string    `json:"role"      example:"member" enums:"owner,member"`
	JoinedAt time.Time `json:"joined_at"`
}

type ListSummaryDTOResponse struct {
	ListDTOResponse
	OpenTasks  int             `json:"open_tasks"  example:"3"`
	TotalTasks int             `json:"total_tasks" example:"5"`
	Role       string          `json:"role"        example:"owner" enums:"owner,member"`
	Members    []ListMemberDTO `json:"members"`
}

// listDTO показывает приглашение только тому, кто может им управлять.
func (h *ListsHTTPHandler) listDTO(l domain.List, canManage bool) ListDTOResponse {
	dto := ListDTOResponse{
		ID:          l.ID,
		Version:     l.Version,
		Title:       l.Title,
		Color:       l.Color,
		OwnerUserID: l.OwnerUserID,
		IsDefault:   l.IsDefault,
		CreatedAt:   l.CreatedAt,
	}

	if canManage && l.InviteCode != nil {
		dto.InviteCode = l.InviteCode
		if h.botUsername != "" {
			link := "https://t.me/" + h.botUsername + "?startapp=join_" + *l.InviteCode
			dto.InviteLink = &link
		}
	}

	return dto
}

func (h *ListsHTTPHandler) listSummaryDTO(s domain.ListSummary) ListSummaryDTOResponse {
	members := make([]ListMemberDTO, len(s.Members))
	for i, m := range s.Members {
		members[i] = ListMemberDTO{UserID: m.UserID, FullName: m.FullName, Role: m.Role, JoinedAt: m.JoinedAt}
	}

	return ListSummaryDTOResponse{
		ListDTOResponse: h.listDTO(s.List, s.Role == domain.ListRoleOwner),
		OpenTasks:       s.OpenTasks,
		TotalTasks:      s.TotalTasks,
		Role:            s.Role,
		Members:         members,
	}
}
