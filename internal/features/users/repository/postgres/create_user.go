package users_postgres_repository

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (r *UsersRepository) CreateUser(
	ctx context.Context,
	user domain.User,
) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.users (full_name, phone_number, telegram_id)
	VALUES ($1, $2, $3)
	RETURNING ` + userColumns + `;
	`
	row := r.pool.QueryRow(ctx, query, user.FullName, user.PhoneNumber, user.TelegramID)

	userModel, err := scanUserModel(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("scan error: %w", err)
	}

	return userDomainFromModel(userModel), nil
}
