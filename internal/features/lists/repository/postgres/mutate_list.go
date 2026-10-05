package lists_postgres_repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
)

func (r *ListsRepository) CreateList(ctx context.Context, list domain.List) (domain.List, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.lists (title, color, owner_user_id)
	VALUES ($1, $2, $3)
	RETURNING ` + listColumns + `;
	`

	m, err := scanListModel(r.pool.QueryRow(ctx, query, list.Title, list.Color, list.OwnerUserID))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrViolatesForeignKey) {
			return domain.List{}, fmt.Errorf("user with id='%d': %w", list.OwnerUserID, core_errors.ErrNotFound)
		}
		return domain.List{}, fmt.Errorf("scan error: %w", err)
	}

	return listDomainFromModel(m), nil
}

func (r *ListsRepository) PatchList(ctx context.Context, id int, list domain.List) (domain.List, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	UPDATE todoapp.lists
	SET title=$1, color=$2, version=version + 1
	WHERE id=$3 AND version=$4
	RETURNING ` + listColumns + `;
	`

	m, err := scanListModel(r.pool.QueryRow(ctx, query, list.Title, list.Color, id, list.Version))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.List{}, fmt.Errorf("list with id='%d' concurrently accessed: %w", id, core_errors.ErrConflict)
		}
		return domain.List{}, fmt.Errorf("scan error: %w", err)
	}

	return listDomainFromModel(m), nil
}

// DeleteList удаляет список вместе с его задачами (ON DELETE CASCADE).
func (r *ListsRepository) DeleteList(ctx context.Context, id int) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tag, err := r.pool.Exec(ctx, `DELETE FROM todoapp.lists WHERE id=$1;`, id)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("list with id='%d': %w", id, core_errors.ErrNotFound)
	}

	return nil
}
