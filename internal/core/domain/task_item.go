package domain

import (
	"fmt"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

const (
	TaskItemTitleMinLen = 1
	TaskItemTitleMaxLen = 200

	// MaxTaskItems — сколько пунктов может быть у одной задачи.
	MaxTaskItems = 100
)

// TaskItem — пункт чеклиста задачи: только текст и галочка.
type TaskItem struct {
	ID        int
	Version   int
	TaskID    int
	Title     string
	Done      bool
	Position  int
	CreatedAt time.Time
}

func NewTaskItemUninitialized(taskID int, title string) TaskItem {
	return TaskItem{
		ID:        UninitializedID,
		Version:   UninitializedVersion,
		TaskID:    taskID,
		Title:     title,
		CreatedAt: time.Now(),
	}
}

func (i *TaskItem) Validate() error {
	titleLen := len([]rune(i.Title))
	if titleLen < TaskItemTitleMinLen || titleLen > TaskItemTitleMaxLen {
		return fmt.Errorf("invalid item `Title` len: %d: %w", titleLen, core_errors.ErrInvalidArgument)
	}
	return nil
}

type TaskItemPatch struct {
	Title Nullable[string]
	Done  Nullable[bool]

	// ExpectedVersion — версия, которую видел клиент; при расхождении — ErrConflict.
	ExpectedVersion *int
}

func (p *TaskItemPatch) Validate() error {
	if p.Title.Set && p.Title.Value == nil {
		return fmt.Errorf("'Title' can't be patched to NULL: %w", core_errors.ErrInvalidArgument)
	}
	if p.Done.Set && p.Done.Value == nil {
		return fmt.Errorf("'Done' can't be patched to NULL: %w", core_errors.ErrInvalidArgument)
	}
	return nil
}

func (i *TaskItem) ApplyPatch(patch TaskItemPatch) error {
	if err := patch.Validate(); err != nil {
		return fmt.Errorf("validate item patch: %w", err)
	}

	if err := checkExpectedVersion(patch.ExpectedVersion, i.Version); err != nil {
		return err
	}

	tmp := *i

	if patch.Title.Set {
		tmp.Title = *patch.Title.Value
	}
	if patch.Done.Set {
		tmp.Done = *patch.Done.Value
	}

	if err := tmp.Validate(); err != nil {
		return fmt.Errorf("validate patched item: %w", err)
	}

	*i = tmp
	return nil
}
