package lists_postgres_repository

import (
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

const listColumns = `id, version, title, color, owner_user_id, is_default, created_at`

type ListModel struct {
	ID          int
	Version     int
	Title       string
	Color       string
	OwnerUserID int
	IsDefault   bool
	CreatedAt   time.Time
}

type scanner interface {
	Scan(dest ...any) error
}

func listScanTargets(m *ListModel) []any {
	return []any{&m.ID, &m.Version, &m.Title, &m.Color, &m.OwnerUserID, &m.IsDefault, &m.CreatedAt}
}

func scanListModel(s scanner) (ListModel, error) {
	var m ListModel
	err := s.Scan(listScanTargets(&m)...)
	return m, err
}

func listDomainFromModel(m ListModel) domain.List {
	return domain.NewList(m.ID, m.Version, m.Title, m.Color, m.OwnerUserID, m.IsDefault, m.CreatedAt)
}
