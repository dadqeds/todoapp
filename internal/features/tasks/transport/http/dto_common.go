package tasks_transport_http

import (
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

type TaskDTOResponse struct {
	ID           int        `json:"id"`
	Version      int        `json:"version"`
	Title        string     `json:"title"`
	Description  *string    `json:"description"`
	Completed    bool       `json:"completed"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	AuthorUserID int        `json:"author_user_id"`
	ListID       int        `json:"list_id"`
	DueAt        *time.Time `json:"due_at"`
	DueAllDay    bool       `json:"due_all_day"`
	Repeat       *RepeatDTO `json:"repeat"`
}

// RepeatDTO — правило повтора. weekdays нужны только для weekly (1 = пн … 7 = вс);
// день месяца и года берутся из срока задачи.
type RepeatDTO struct {
	Kind     string `json:"kind"               example:"weekly" enums:"daily,weekly,monthly,yearly"`
	Weekdays []int  `json:"weekdays,omitempty" example:"1,4"`
}

func repeatDTOFromDomain(r *domain.Recurrence) *RepeatDTO {
	if r == nil {
		return nil
	}
	dto := &RepeatDTO{Kind: string(r.Kind)}
	if r.Kind == domain.RepeatWeekly {
		dto.Weekdays = r.Weekdays
	}
	return dto
}

func (d *RepeatDTO) toDomain() *domain.Recurrence {
	if d == nil {
		return nil
	}
	return &domain.Recurrence{Kind: domain.RepeatKind(d.Kind), Weekdays: d.Weekdays}
}

func taskDTOFromDomain(task domain.Task) TaskDTOResponse {
	return TaskDTOResponse{
		ID:           task.ID,
		Version:      task.Version,
		Title:        task.Title,
		Description:  task.Description,
		Completed:    task.Completed,
		CreatedAt:    task.CreatedAt,
		CompletedAt:  task.CompletedAt,
		AuthorUserID: task.AuthorUserID,
		ListID:       task.ListID,
		DueAt:        task.DueAt,
		DueAllDay:    task.DueAllDay,
		Repeat:       repeatDTOFromDomain(task.Repeat),
	}
}

func taskDTOFromDomains(tasks []domain.Task) []TaskDTOResponse {
	dtos := make([]TaskDTOResponse, len(tasks))

	for i, task := range tasks {
		dtos[i] = taskDTOFromDomain(task)
	}

	return dtos
}
