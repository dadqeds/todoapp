package core_auth

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// Actor — аутентифицированный пользователь, от имени которого выполняется запрос.
type Actor struct {
	User    domain.User
	IsAdmin bool
}

// CanAccessUser: обычный пользователь работает только со своими данными,
// администратор — со всеми.
func (a Actor) CanAccessUser(userID int) bool {
	return a.IsAdmin || a.User.ID == userID
}

type actorContextKey struct{}

func ToContext(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func FromContext(ctx context.Context) (Actor, error) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)
	if !ok {
		return Actor{}, fmt.Errorf("no authenticated user in context: %w", core_errors.ErrUnauthenticated)
	}
	return actor, nil
}
