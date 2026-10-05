package domain

import (
	"fmt"
	"slices"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

const (
	ListTitleMinLen = 1
	ListTitleMaxLen = 50

	DefaultListTitle = "Личное"
	DefaultListColor = "coral"
)

// ListColors — палитра списков. Конкретные оттенки задаёт фронтенд.
var ListColors = []string{"green", "violet", "coral", "blue", "pink", "amber"}

type List struct {
	ID          int
	Version     int
	Title       string
	Color       string
	OwnerUserID int
	IsDefault   bool
	CreatedAt   time.Time
}

func NewList(
	id int,
	version int,
	title string,
	color string,
	ownerUserID int,
	isDefault bool,
	createdAt time.Time,
) List {
	return List{
		ID:          id,
		Version:     version,
		Title:       title,
		Color:       color,
		OwnerUserID: ownerUserID,
		IsDefault:   isDefault,
		CreatedAt:   createdAt,
	}
}

func NewListUninitialized(title string, color string, ownerUserID int) List {
	return NewList(UninitializedID, UninitializedVersion, title, color, ownerUserID, false, time.Now())
}

func (l *List) Validate() error {
	titleLen := len([]rune(l.Title))
	if titleLen < ListTitleMinLen || titleLen > ListTitleMaxLen {
		return fmt.Errorf("invalid list `Title` len: %d: %w", titleLen, core_errors.ErrInvalidArgument)
	}

	if !slices.Contains(ListColors, l.Color) {
		return fmt.Errorf("invalid list `Color` %q: %w", l.Color, core_errors.ErrInvalidArgument)
	}

	return nil
}

type ListPatch struct {
	Title           Nullable[string]
	Color           Nullable[string]
	ExpectedVersion *int
}

func (p *ListPatch) Validate() error {
	if p.Title.Set && p.Title.Value == nil {
		return fmt.Errorf("'Title' can't be patched to NULL: %w", core_errors.ErrInvalidArgument)
	}
	if p.Color.Set && p.Color.Value == nil {
		return fmt.Errorf("'Color' can't be patched to NULL: %w", core_errors.ErrInvalidArgument)
	}
	return nil
}

func (l *List) ApplyPatch(patch ListPatch) error {
	if err := patch.Validate(); err != nil {
		return fmt.Errorf("validate list patch: %w", err)
	}

	if err := checkExpectedVersion(patch.ExpectedVersion, l.Version); err != nil {
		return err
	}

	tmp := *l

	if patch.Title.Set {
		tmp.Title = *patch.Title.Value
	}
	if patch.Color.Set {
		tmp.Color = *patch.Color.Value
	}

	if err := tmp.Validate(); err != nil {
		return fmt.Errorf("validate patched list: %w", err)
	}

	*l = tmp
	return nil
}

// ListSummary — список вместе со счётчиками задач для экрана списков.
type ListSummary struct {
	List
	OpenTasks  int
	TotalTasks int
}
