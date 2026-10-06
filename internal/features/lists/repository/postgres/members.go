package lists_postgres_repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
)

func (r *ListsRepository) IsListMember(ctx context.Context, listID int, userID int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM todoapp.list_members WHERE list_id=$1 AND user_id=$2);`
	if err := r.pool.QueryRow(ctx, query, listID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("select list member: %w", err)
	}

	return exists, nil
}

func (r *ListsRepository) AddMember(ctx context.Context, listID int, userID int) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.list_members (list_id, user_id)
	VALUES ($1, $2)
	ON CONFLICT (list_id, user_id) DO NOTHING;
	`
	if _, err := r.pool.Exec(ctx, query, listID, userID); err != nil {
		return fmt.Errorf("insert list member: %w", err)
	}

	return nil
}

func (r *ListsRepository) RemoveMember(ctx context.Context, listID int, userID int) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tag, err := r.pool.Exec(ctx, `DELETE FROM todoapp.list_members WHERE list_id=$1 AND user_id=$2;`, listID, userID)
	if err != nil {
		return fmt.Errorf("delete list member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user with id='%d' in list with id='%d': %w", userID, listID, core_errors.ErrNotFound)
	}

	return nil
}

// SetInviteCode включает (code != nil) или выключает приглашение по ссылке.
func (r *ListsRepository) SetInviteCode(ctx context.Context, listID int, code *string) (domain.List, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	UPDATE todoapp.lists
	SET invite_code=$1
	WHERE id=$2
	RETURNING ` + listColumns + `;
	`

	m, err := scanListModel(r.pool.QueryRow(ctx, query, code, listID))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.List{}, fmt.Errorf("list with id='%d': %w", listID, core_errors.ErrNotFound)
		}
		return domain.List{}, fmt.Errorf("scan error: %w", err)
	}

	return listDomainFromModel(m), nil
}

func (r *ListsRepository) GetListByInviteCode(ctx context.Context, code string) (domain.List, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT ` + listColumns + ` FROM todoapp.lists WHERE invite_code=$1;`

	m, err := scanListModel(r.pool.QueryRow(ctx, query, code))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.List{}, fmt.Errorf("invite code: %w", core_errors.ErrNotFound)
		}
		return domain.List{}, fmt.Errorf("scan error: %w", err)
	}

	return listDomainFromModel(m), nil
}

// SetNotifyChanges включает или выключает уведомления об изменениях в списке
// для пользователя: у владельца флаг хранится в самом списке, у участника — в
// list_members. Не владелец и не участник — ErrNotFound.
func (r *ListsRepository) SetNotifyChanges(ctx context.Context, listID int, userID int, enabled bool) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	WITH owner AS (
		UPDATE todoapp.lists SET owner_notify_changes = $3
		WHERE id = $1 AND owner_user_id = $2
		RETURNING id
	), member AS (
		UPDATE todoapp.list_members SET notify_changes = $3
		WHERE list_id = $1 AND user_id = $2
		RETURNING list_id
	)
	SELECT (SELECT COUNT(*) FROM owner) + (SELECT COUNT(*) FROM member);
	`

	var updated int
	if err := r.pool.QueryRow(ctx, query, listID, userID, enabled).Scan(&updated); err != nil {
		return fmt.Errorf("update notify changes: %w", err)
	}
	if updated == 0 {
		return fmt.Errorf("user with id='%d' in list with id='%d': %w", userID, listID, core_errors.ErrNotFound)
	}

	return nil
}
