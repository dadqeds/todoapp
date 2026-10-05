package users_service

import (
	"context"
	"errors"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// ResolveTelegramUser возвращает пользователя, привязанного к аккаунту Telegram,
// и создаёт его при первом входе. Имя берётся из Telegram только при создании,
// дальше пользователь может менять его сам.
func (s *UsersService) ResolveTelegramUser(
	ctx context.Context,
	tgUser core_auth.TelegramUser,
) (domain.User, error) {
	user, err := s.usersRepository.GetUserByTelegramID(ctx, tgUser.ID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		return domain.User{}, fmt.Errorf("get user by telegram id: %w", err)
	}

	user = domain.NewTelegramUserUninitialized(tgUser.FullName(), tgUser.ID)
	if err := user.Validate(); err != nil {
		return domain.User{}, fmt.Errorf("validate telegram user: %w", err)
	}

	user, err = s.usersRepository.CreateTelegramUser(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("create telegram user: %w", err)
	}

	return user, nil
}
