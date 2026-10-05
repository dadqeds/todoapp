package lists_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// GetLists возвращает свои и общие списки пользователя. Администратор может
// запросить списки другого пользователя.
func (s *ListsService) GetLists(ctx context.Context, userID *int) ([]domain.ListSummary, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	owner := actor.User.ID
	if actor.IsAdmin && userID != nil {
		owner = *userID
	}

	// Гарантируем, что у пользователя есть список по умолчанию.
	if _, err := s.listsRepository.GetOrCreateDefaultList(ctx, owner); err != nil {
		return nil, fmt.Errorf("ensure default list: %w", err)
	}

	lists, err := s.listsRepository.GetListsForUser(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("get lists from repository: %w", err)
	}

	return lists, nil
}

// getListForView: список видят владелец, участники и администратор.
// Остальным он отдаётся как несуществующий.
func (s *ListsService) getListForView(ctx context.Context, id int) (domain.List, core_auth.Actor, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.List{}, actor, err
	}

	list, err := s.listsRepository.GetList(ctx, id)
	if err != nil {
		return domain.List{}, actor, fmt.Errorf("get list from repository: %w", err)
	}

	if actor.CanAccessUser(list.OwnerUserID) {
		return list, actor, nil
	}

	member, err := s.listsRepository.IsListMember(ctx, id, actor.User.ID)
	if err != nil {
		return domain.List{}, actor, fmt.Errorf("check list member: %w", err)
	}
	if !member {
		return domain.List{}, actor, fmt.Errorf("list with id='%d': %w", id, core_errors.ErrNotFound)
	}

	return list, actor, nil
}

// getListForManage: менять список, приглашать и исключать может только владелец
// (и администратор). Участник получает 403 — список он видит.
func (s *ListsService) getListForManage(ctx context.Context, id int) (domain.List, error) {
	list, actor, err := s.getListForView(ctx, id)
	if err != nil {
		return domain.List{}, err
	}

	if !actor.CanAccessUser(list.OwnerUserID) {
		return domain.List{}, fmt.Errorf("only owner can manage list with id='%d': %w", id, core_errors.ErrForbidden)
	}

	return list, nil
}

func (s *ListsService) CreateList(ctx context.Context, list domain.List) (domain.List, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.List{}, err
	}

	list.OwnerUserID = actor.User.ID
	list.IsDefault = false

	if err := list.Validate(); err != nil {
		return domain.List{}, fmt.Errorf("validate list: %w", err)
	}

	list, err = s.listsRepository.CreateList(ctx, list)
	if err != nil {
		return domain.List{}, fmt.Errorf("create list: %w", err)
	}

	return list, nil
}

func (s *ListsService) PatchList(ctx context.Context, id int, patch domain.ListPatch) (domain.List, error) {
	list, err := s.getListForManage(ctx, id)
	if err != nil {
		return domain.List{}, err
	}

	if err := list.ApplyPatch(patch); err != nil {
		return domain.List{}, fmt.Errorf("apply list patch: %w", err)
	}

	patched, err := s.listsRepository.PatchList(ctx, id, list)
	if err != nil {
		return domain.List{}, fmt.Errorf("patch list: %w", err)
	}

	return patched, nil
}

// DeleteList удаляет список вместе с задачами. Список по умолчанию удалить нельзя.
func (s *ListsService) DeleteList(ctx context.Context, id int) error {
	list, err := s.getListForManage(ctx, id)
	if err != nil {
		return err
	}

	if list.IsDefault {
		return fmt.Errorf("default list can't be deleted: %w", core_errors.ErrConflict)
	}

	if err := s.listsRepository.DeleteList(ctx, id); err != nil {
		return fmt.Errorf("delete list: %w", err)
	}

	return nil
}
