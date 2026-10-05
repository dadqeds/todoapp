package lists_postgres_repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
)

func (r *ListsRepository) GetList(ctx context.Context, id int) (domain.List, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT ` + listColumns + ` FROM todoapp.lists WHERE id=$1;`

	m, err := scanListModel(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.List{}, fmt.Errorf("list with id='%d': %w", id, core_errors.ErrNotFound)
		}
		return domain.List{}, fmt.Errorf("scan error: %w", err)
	}

	return listDomainFromModel(m), nil
}

// GetOrCreateDefaultList возвращает список по умолчанию пользователя и создаёт
// его, если его ещё нет (новые пользователи, гонка двух первых запросов).
func (r *ListsRepository) GetOrCreateDefaultList(ctx context.Context, ownerUserID int) (domain.List, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	insert := `
	INSERT INTO todoapp.lists (title, color, owner_user_id, is_default)
	VALUES ($1, $2, $3, TRUE)
	ON CONFLICT (owner_user_id) WHERE is_default DO NOTHING;
	`
	if _, err := r.pool.Exec(ctx, insert, domain.DefaultListTitle, domain.DefaultListColor, ownerUserID); err != nil {
		return domain.List{}, fmt.Errorf("insert default list: %w", err)
	}

	query := `SELECT ` + listColumns + ` FROM todoapp.lists WHERE owner_user_id=$1 AND is_default;`

	m, err := scanListModel(r.pool.QueryRow(ctx, query, ownerUserID))
	if err != nil {
		return domain.List{}, fmt.Errorf("select default list: %w", err)
	}

	return listDomainFromModel(m), nil
}
