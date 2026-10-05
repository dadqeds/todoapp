package users_postgres_repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
)

func (r *UsersRepository) GetUserByTelegramID(
	ctx context.Context,
	telegramID int64,
) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT ` + userColumns + `
	FROM todoapp.users
	WHERE telegram_id=$1;
	`

	userModel, err := scanUserModel(r.pool.QueryRow(ctx, query, telegramID))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.User{}, fmt.Errorf(
				"user with telegram_id='%d': %w",
				telegramID,
				core_errors.ErrNotFound,
			)
		}
		return domain.User{}, fmt.Errorf("scan error: %w", err)
	}

	return userDomainFromModel(userModel), nil
}

// CreateTelegramUser создаёт пользователя, если пользователя с таким
// telegram_id ещё нет. При гонке двух первых запросов возвращает того,
// кого успел создать конкурент.
func (r *UsersRepository) CreateTelegramUser(
	ctx context.Context,
	user domain.User,
) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.users (full_name, telegram_id)
	VALUES ($1, $2)
	ON CONFLICT (telegram_id) DO NOTHING
	RETURNING ` + userColumns + `;
	`

	userModel, err := scanUserModel(r.pool.QueryRow(ctx, query, user.FullName, user.TelegramID))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return r.GetUserByTelegramID(ctx, *user.TelegramID)
		}
		return domain.User{}, fmt.Errorf("scan error: %w", err)
	}

	return userDomainFromModel(userModel), nil
}
