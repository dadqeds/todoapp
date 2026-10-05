package lists_postgres_repository

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

// GetLists возвращает списки пользователя (или все, если ownerUserID=nil)
// со счётчиками задач. Список по умолчанию идёт первым.
func (r *ListsRepository) GetLists(ctx context.Context, ownerUserID *int) ([]domain.ListSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT l.id, l.version, l.title, l.color, l.owner_user_id, l.is_default, l.created_at,
		COUNT(t.id) FILTER (WHERE NOT t.completed),
		COUNT(t.id)
	FROM todoapp.lists l
	LEFT JOIN todoapp.tasks t ON t.list_id = l.id
	WHERE $1::int IS NULL OR l.owner_user_id = $1
	GROUP BY l.id
	ORDER BY l.is_default DESC, l.id ASC;
	`

	rows, err := r.pool.Query(ctx, query, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("select lists: %w", err)
	}
	defer rows.Close()

	var lists []domain.ListSummary
	for rows.Next() {
		var (
			m           ListModel
			open, total int
		)
		if err := rows.Scan(append(listScanTargets(&m), &open, &total)...); err != nil {
			return nil, fmt.Errorf("scan lists: %w", err)
		}
		lists = append(lists, domain.ListSummary{List: listDomainFromModel(m), OpenTasks: open, TotalTasks: total})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return lists, nil
}
