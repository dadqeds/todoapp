package tasks_postgres_repository

import (
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

// Последние два столбца — счётчики пунктов чеклиста; запросы обращаются к
// таблице задач по имени tasks, без псевдонима.
const taskColumns = `id, version, title, description, completed, created_at, completed_at, author_user_id, list_id, due_at, due_all_day, repeat_rule, remind_before_minutes, reminded_at,
	(SELECT COUNT(*) FROM todoapp.task_items i WHERE i.task_id = tasks.id),
	(SELECT COUNT(*) FROM todoapp.task_items i WHERE i.task_id = tasks.id AND i.done)`

type TaskModel struct {
	ID           int
	Version      int
	Title        string
	Description  *string
	Completed    bool
	CreatedAt    time.Time
	CompletedAt  *time.Time
	AuthorUserID int
	ListID       int
	DueAt        *time.Time
	DueAllDay    bool
	RepeatRule   *string
	RemindBefore *int
	RemindedAt   *time.Time
	ItemsTotal   int
	ItemsDone    int
}

// scanner покрывает и core_postgres_pool.Row, и core_postgres_pool.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTaskModel(s scanner) (TaskModel, error) {
	var taskModel TaskModel

	err := s.Scan(
		&taskModel.ID,
		&taskModel.Version,
		&taskModel.Title,
		&taskModel.Description,
		&taskModel.Completed,
		&taskModel.CreatedAt,
		&taskModel.CompletedAt,
		&taskModel.AuthorUserID,
		&taskModel.ListID,
		&taskModel.DueAt,
		&taskModel.DueAllDay,
		&taskModel.RepeatRule,
		&taskModel.RemindBefore,
		&taskModel.RemindedAt,
		&taskModel.ItemsTotal,
		&taskModel.ItemsDone,
	)

	return taskModel, err
}

func taskDomainFromModel(taskModel TaskModel) domain.Task {
	task := domain.NewTask(
		taskModel.ID,
		taskModel.Version,
		taskModel.Title,
		taskModel.Description,
		taskModel.Completed,
		taskModel.CreatedAt,
		taskModel.CompletedAt,
		taskModel.AuthorUserID,
		taskModel.ListID,
		taskModel.DueAt,
		taskModel.DueAllDay,
	)

	// Правило проверяется при записи; если в БД оказалось что-то неразборчивое,
	// задача просто считается неповторяющейся.
	if taskModel.RepeatRule != nil {
		if repeat, err := domain.ParseRecurrence(*taskModel.RepeatRule); err == nil {
			task.Repeat = &repeat
		}
	}

	task.RemindBeforeMinutes = taskModel.RemindBefore
	task.RemindedAt = taskModel.RemindedAt
	task.ItemsTotal = taskModel.ItemsTotal
	task.ItemsDone = taskModel.ItemsDone

	return task
}

// repeatRuleToModel — правило в формате хранения или NULL.
func repeatRuleToModel(repeat *domain.Recurrence) *string {
	if repeat == nil {
		return nil
	}
	rule := repeat.String()
	return &rule
}

func taskDomainsFromModels(taskModels []TaskModel) []domain.Task {
	domains := make([]domain.Task, len(taskModels))

	for i, model := range taskModels {
		domains[i] = taskDomainFromModel(model)
	}
	return domains
}
