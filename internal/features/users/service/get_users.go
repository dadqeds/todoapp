package users_service

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (s *UsersService) GetUsers(
	ctx context.Context,
	limit *int,
	offset *int,
) ([]domain.User, error) {
	l, o, err := domain.NormalizePagination(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("normalize pagination: %w", err)
	}

	users, err := s.usersRepository.GetUsers(ctx, l, o)
	if err != nil {
		return nil, fmt.Errorf(
			"get users from repository: %w",
			err,
		)
	}
	return users, nil
}
