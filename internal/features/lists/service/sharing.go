package lists_service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// newInviteCode — 16 случайных символов [A-Za-z0-9_-]: подходит для
// параметра startapp ссылки Telegram (до 64 символов из этого алфавита).
func newInviteCode() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateInvite выдаёт новую ссылку-приглашение. Прежняя ссылка перестаёт работать.
func (s *ListsService) CreateInvite(ctx context.Context, listID int) (domain.List, error) {
	list, err := s.getListForManage(ctx, listID)
	if err != nil {
		return domain.List{}, err
	}

	if list.IsDefault {
		return domain.List{}, fmt.Errorf("default list can't be shared: %w", core_errors.ErrConflict)
	}

	code, err := newInviteCode()
	if err != nil {
		return domain.List{}, err
	}

	list, err = s.listsRepository.SetInviteCode(ctx, listID, &code)
	if err != nil {
		return domain.List{}, fmt.Errorf("set invite code: %w", err)
	}

	return list, nil
}

// RevokeInvite выключает приглашение по ссылке. Уже вступившие остаются.
func (s *ListsService) RevokeInvite(ctx context.Context, listID int) error {
	if _, err := s.getListForManage(ctx, listID); err != nil {
		return err
	}

	if _, err := s.listsRepository.SetInviteCode(ctx, listID, nil); err != nil {
		return fmt.Errorf("revoke invite code: %w", err)
	}

	return nil
}

// JoinList добавляет текущего пользователя в список по коду приглашения.
// Повторное вступление и вступление владельца ничего не меняют.
func (s *ListsService) JoinList(ctx context.Context, code string) (domain.List, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.List{}, err
	}

	list, err := s.listsRepository.GetListByInviteCode(ctx, code)
	if err != nil {
		return domain.List{}, fmt.Errorf("find list by invite: %w", err)
	}

	if list.OwnerUserID == actor.User.ID {
		return list, nil
	}

	if err := s.listsRepository.AddMember(ctx, list.ID, actor.User.ID); err != nil {
		return domain.List{}, fmt.Errorf("add member: %w", err)
	}

	return list, nil
}

// RemoveMember: участник может выйти сам, владелец — исключить любого участника.
// Владелец не может выйти из своего списка: его можно только удалить.
func (s *ListsService) RemoveMember(ctx context.Context, listID int, userID int) error {
	list, actor, err := s.getListForView(ctx, listID)
	if err != nil {
		return err
	}

	if userID == list.OwnerUserID {
		return fmt.Errorf("owner can't leave own list: %w", core_errors.ErrConflict)
	}

	if userID != actor.User.ID && !actor.CanAccessUser(list.OwnerUserID) {
		return fmt.Errorf("only owner can remove members: %w", core_errors.ErrForbidden)
	}

	if err := s.listsRepository.RemoveMember(ctx, listID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}
