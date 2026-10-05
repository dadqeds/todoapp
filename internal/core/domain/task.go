package domain

import (
	"fmt"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type Task struct {
	ID      int
	Version int

	Title       string
	Description *string
	Completed   bool
	CreatedAt   time.Time
	CompletedAt *time.Time

	AuthorUserID int
	ListID       int

	// DueAt — срок. При DueAllDay=true это конец дня по времени пользователя,
	// и показывать нужно только дату.
	DueAt     *time.Time
	DueAllDay bool

	// Repeat — правило повтора; требует срока.
	Repeat *Recurrence
}

func NewTask(
	id int,
	version int,
	title string,
	description *string,
	completed bool,
	createdAt time.Time,
	completedAt *time.Time,
	authorUserID int,
	listID int,
	dueAt *time.Time,
	dueAllDay bool,
) Task {
	return Task{
		ID:           id,
		Version:      version,
		Title:        title,
		Description:  description,
		Completed:    completed,
		CreatedAt:    createdAt,
		CompletedAt:  completedAt,
		AuthorUserID: authorUserID,
		ListID:       listID,
		DueAt:        dueAt,
		DueAllDay:    dueAllDay,
	}
}

// NewTaskUninitialized создаёт новую задачу. listID=0 означает «список по
// умолчанию автора»: его подставляет сервис.
func NewTaskUninitialized(
	title string,
	description *string,
	authorUserID int,
	listID int,
	dueAt *time.Time,
	dueAllDay bool,
) Task {
	return NewTask(
		UninitializedID,
		UninitializedVersion,
		title,
		description,
		false,
		time.Now(),
		nil,
		authorUserID,
		listID,
		dueAt,
		dueAllDay,
	)
}

// IsCompletedOnTime: задача выполнена и не позже срока.
func (t *Task) IsCompletedOnTime() bool {
	return t.Completed && t.CompletedAt != nil && t.DueAt != nil && !t.CompletedAt.After(*t.DueAt)
}

func (t *Task) CompletionDuration() *time.Duration {
	if !t.Completed {
		return nil
	}

	if t.CompletedAt == nil {
		return nil
	}

	duration := t.CompletedAt.Sub(t.CreatedAt)
	return &duration
}

func (t *Task) Validate() error {
	titleLen := len([]rune(t.Title))
	if titleLen < 1 || titleLen > 100 {
		return fmt.Errorf(
			"invalid 'Title' len: %d: %w",
			titleLen,
			core_errors.ErrInvalidArgument,
		)
	}

	if t.Description != nil {
		descriptionLen := len([]rune(*t.Description))
		if descriptionLen < 1 || descriptionLen > 1000 {
			return fmt.Errorf(
				"invalid 'Description' len: %d: %w",
				descriptionLen,
				core_errors.ErrInvalidArgument,
			)
		}
	}

	if t.DueAt == nil && t.DueAllDay {
		return fmt.Errorf(
			"`DueAllDay` requires `DueAt`: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	if t.Repeat != nil {
		if t.DueAt == nil {
			return fmt.Errorf(
				"`Repeat` requires `DueAt`: %w",
				core_errors.ErrInvalidArgument,
			)
		}
		if err := t.Repeat.Validate(); err != nil {
			return fmt.Errorf("validate repeat: %w", err)
		}
	}

	if t.Completed {
		if t.CompletedAt == nil {
			return fmt.Errorf(
				"`CompletedAt` can't be `nil` if `Completed`==`true`: %w",
				core_errors.ErrInvalidArgument,
			)
		}

		if t.CompletedAt.Before(t.CreatedAt) {
			return fmt.Errorf(
				"`CompletedAt` can't be before `CreatedAt`: %w",
				core_errors.ErrInvalidArgument,
			)
		}
	} else {
		if t.CompletedAt != nil {
			return fmt.Errorf(
				"`CompletedAt` must be `nil` if `Completed`==`false`: %w",
				core_errors.ErrInvalidArgument,
			)
		}
	}

	return nil
}

type TaskPatch struct {
	Title       Nullable[string]
	Description Nullable[string]
	Completed   Nullable[bool]
	ListID      Nullable[int]

	// DueAt=null снимает срок (и DueAllDay вместе с ним).
	DueAt     Nullable[time.Time]
	DueAllDay Nullable[bool]

	// Repeat=null выключает повтор. Снятие срока выключает повтор тоже.
	Repeat Nullable[Recurrence]

	// ExpectedVersion — версия, которую видел клиент. Если задана и не совпадает
	// с текущей, патч отклоняется с ErrConflict (оптимистичная блокировка).
	ExpectedVersion *int
}

func (p *TaskPatch) Validate() error {
	if p.Title.Set && p.Title.Value == nil {
		return fmt.Errorf(
			"'Title' can't be patched to NULL: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	if p.ListID.Set && p.ListID.Value == nil {
		return fmt.Errorf(
			"'ListID' can't be patched to NULL: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	if p.DueAllDay.Set && p.DueAllDay.Value == nil {
		return fmt.Errorf(
			"'DueAllDay' can't be patched to NULL: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	if p.Completed.Set && p.Completed.Value == nil {
		return fmt.Errorf(
			"'Completed' can't be patched to NULL: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	return nil
}

func (t *Task) ApplyPatch(patch TaskPatch) error {
	if err := patch.Validate(); err != nil {
		return fmt.Errorf("validate task patch: %w", err)
	}

	if err := checkExpectedVersion(patch.ExpectedVersion, t.Version); err != nil {
		return err
	}

	tmp := *t

	if patch.Title.Set {
		tmp.Title = *patch.Title.Value
	}

	if patch.Description.Set {
		tmp.Description = patch.Description.Value
	}

	if patch.ListID.Set {
		tmp.ListID = *patch.ListID.Value
	}

	if patch.DueAt.Set {
		tmp.DueAt = patch.DueAt.Value
		if tmp.DueAt == nil {
			tmp.DueAllDay = false
			tmp.Repeat = nil
		}
	}

	if patch.DueAllDay.Set {
		tmp.DueAllDay = *patch.DueAllDay.Value
	}

	if patch.Repeat.Set {
		tmp.Repeat = patch.Repeat.Value
	}

	if patch.Completed.Set {
		wasCompleted := tmp.Completed
		tmp.Completed = *patch.Completed.Value

		switch {
		case tmp.Completed && !wasCompleted:
			completedAt := time.Now()
			tmp.CompletedAt = &completedAt
		case !tmp.Completed:
			tmp.CompletedAt = nil
		}
	}

	if err := tmp.Validate(); err != nil {
		return fmt.Errorf("validate patched task: %w", err)
	}

	*t = tmp

	return nil
}

// TaskFilter — условия выборки задач; nil-поля не ограничивают выборку.
type TaskFilter struct {
	AuthorUserID *int
	ListID       *int
	// AccessibleToUserID — задачи из списков, которыми пользователь владеет или в которых участвует.
	AccessibleToUserID *int
}

// NextOccurrence возвращает следующий экземпляр повторяющейся задачи: та же
// задача с новым сроком, ещё не выполненная. ok=false, если повтора нет.
func (t *Task) NextOccurrence(now time.Time, loc *time.Location) (next Task, ok bool) {
	if t.Repeat == nil || t.DueAt == nil {
		return Task{}, false
	}

	due := t.Repeat.Next(*t.DueAt, now, loc)
	repeat := *t.Repeat

	next = NewTaskUninitialized(t.Title, t.Description, t.AuthorUserID, t.ListID, &due, t.DueAllDay)
	next.Repeat = &repeat
	return next, true
}
