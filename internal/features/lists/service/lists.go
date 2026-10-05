package lists_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// GetLists возвращает списки пользователя. Обычный пользователь видит только
// свои; администратор может запросить списки другого пользователя.
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

	lists, err := s.listsRepository.GetLists(ctx, &owner)
	if err != nil {
		return nil, fmt.Errorf("get lists from repository: %w", err)
	}

	return lists, nil
}

func (s *ListsService) getAccessibleList(ctx context.Context, id int) (domain.List, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.List{}, err
	}

	list, err := s.listsRepository.GetList(ctx, id)
	if err != nil {
		return domain.List{}, fmt.Errorf("get list from repository: %w", err)
	}

	if !actor.CanAccessUser(list.OwnerUserID) {
		return domain.List{}, fmt.Errorf("list with id='%d': %w", id, core_errors.ErrNotFound)
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
	list, err := s.getAccessibleList(ctx, id)
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
	list, err := s.getAccessibleList(ctx, id)
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
