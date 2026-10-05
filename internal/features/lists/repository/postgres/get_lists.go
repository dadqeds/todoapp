package lists_postgres_repository

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

// GetListsForUser возвращает списки, которыми пользователь владеет или в
// которых участвует, со счётчиками задач, ролью пользователя и участниками.
// Список по умолчанию идёт первым, общие — после своих.
func (r *ListsRepository) GetListsForUser(ctx context.Context, userID int) ([]domain.ListSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT l.id, l.version, l.title, l.color, l.owner_user_id, l.is_default, l.created_at, l.invite_code,
		COUNT(t.id) FILTER (WHERE NOT t.completed),
		COUNT(t.id)
	FROM todoapp.lists l
	LEFT JOIN todoapp.tasks t ON t.list_id = l.id
	WHERE l.owner_user_id = $1
		OR l.id IN (SELECT list_id FROM todoapp.list_members WHERE user_id = $1)
	GROUP BY l.id
	ORDER BY l.is_default DESC, (l.owner_user_id = $1) DESC, l.id ASC;
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("select lists: %w", err)
	}
	defer rows.Close()

	var (
		lists []domain.ListSummary
		ids   []int
	)
	for rows.Next() {
		var (
			m           ListModel
			open, total int
		)
		if err := rows.Scan(append(listScanTargets(&m), &open, &total)...); err != nil {
			return nil, fmt.Errorf("scan lists: %w", err)
		}

		role := domain.ListRoleMember
		if m.OwnerUserID == userID {
			role = domain.ListRoleOwner
		}

		lists = append(lists, domain.ListSummary{List: listDomainFromModel(m), OpenTasks: open, TotalTasks: total, Role: role})
		ids = append(ids, m.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	members, err := r.getMembers(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range lists {
		lists[i].Members = members[lists[i].ID]
	}

	return lists, nil
}

// getMembers возвращает владельца и участников каждого списка: владелец первым.
func (r *ListsRepository) getMembers(ctx context.Context, listIDs []int) (map[int][]domain.ListMember, error) {
	result := make(map[int][]domain.ListMember, len(listIDs))
	if len(listIDs) == 0 {
		return result, nil
	}

	query := `
	SELECT l.id, u.id, u.full_name, 'owner', l.created_at
	FROM todoapp.lists l
	JOIN todoapp.users u ON u.id = l.owner_user_id
	WHERE l.id = ANY($1)
	UNION ALL
	SELECT lm.list_id, u.id, u.full_name, 'member', lm.joined_at
	FROM todoapp.list_members lm
	JOIN todoapp.users u ON u.id = lm.user_id
	WHERE lm.list_id = ANY($1)
	ORDER BY 1, 4 DESC, 5;
	`

	rows, err := r.pool.Query(ctx, query, listIDs)
	if err != nil {
		return nil, fmt.Errorf("select list members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			listID int
			m      domain.ListMember
		)
		if err := rows.Scan(&listID, &m.UserID, &m.FullName, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan list members: %w", err)
		}
		result[listID] = append(result[listID], m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return result, nil
}
