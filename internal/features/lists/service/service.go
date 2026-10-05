package lists_service

import (
	"context"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

type ListsService struct {
	listsRepository ListsRepository
}

type ListsRepository interface {
	GetList(ctx context.Context, id int) (domain.List, error)
	GetListsForUser(ctx context.Context, userID int) ([]domain.ListSummary, error)
	GetOrCreateDefaultList(ctx context.Context, ownerUserID int) (domain.List, error)
	CreateList(ctx context.Context, list domain.List) (domain.List, error)
	PatchList(ctx context.Context, id int, list domain.List) (domain.List, error)
	DeleteList(ctx context.Context, id int) error

	IsListMember(ctx context.Context, listID int, userID int) (bool, error)
	AddMember(ctx context.Context, listID int, userID int) error
	RemoveMember(ctx context.Context, listID int, userID int) error
	SetInviteCode(ctx context.Context, listID int, code *string) (domain.List, error)
	GetListByInviteCode(ctx context.Context, code string) (domain.List, error)
}

func NewListsService(listsRepository ListsRepository) *ListsService {
	return &ListsService{
		listsRepository: listsRepository,
	}
}
