package users_postgres_repository

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (r *UsersRepository) GetUsers(
	ctx context.Context,
	limit int,
	offset int,
) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT ` + userColumns + `
	FROM todoapp.users
	ORDER BY id ASC
	LIMIT $1
	OFFSET $2;
	`
	rows, err := r.pool.Query(
		ctx,
		query,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	var userModels []UserModel

	for rows.Next() {
		userModel, err := scanUserModel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan users: %w", err)
		}

		userModels = append(userModels, userModel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}
	userdomains := userDomainsFromModels(userModels)

	return userdomains, nil
}
