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
	GetLists(ctx context.Context, ownerUserID *int) ([]domain.ListSummary, error)
	GetOrCreateDefaultList(ctx context.Context, ownerUserID int) (domain.List, error)
	CreateList(ctx context.Context, list domain.List) (domain.List, error)
	PatchList(ctx context.Context, id int, list domain.List) (domain.List, error)
	DeleteList(ctx context.Context, id int) error
}

func NewListsService(listsRepository ListsRepository) *ListsService {
	return &ListsService{
		listsRepository: listsRepository,
	}
}
