package users_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func requireAdmin(ctx context.Context) error {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return err
	}
	if !actor.IsAdmin {
		return fmt.Errorf("admin role required: %w", core_errors.ErrForbidden)
	}
	return nil
}

// requireUserAccess скрывает чужих пользователей за 404, чтобы не раскрывать,
// существует ли запись.
func requireUserAccess(ctx context.Context, userID int) error {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return err
	}
	if !actor.CanAccessUser(userID) {
		return fmt.Errorf("user with id='%d': %w", userID, core_errors.ErrNotFound)
	}
	return nil
}
