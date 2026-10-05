package tasks_service

import (
	"context"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

type TasksService struct {
	tasksRepository TasksRepository
	listsRepository ListsRepository
}

// ListsRepository — то, что сервису задач нужно знать о списках.
type ListsRepository interface {
	GetList(ctx context.Context, id int) (domain.List, error)
	GetOrCreateDefaultList(ctx context.Context, ownerUserID int) (domain.List, error)
	IsListMember(ctx context.Context, listID int, userID int) (bool, error)
}

type TasksRepository interface {
	CreateTask(
		ctx context.Context,
		task domain.Task,
	) (domain.Task, error)

	GetTasks(
		ctx context.Context,
		filter domain.TaskFilter,
		limit int,
		offset int,
	) ([]domain.Task, error)

	GetTask(
		ctx context.Context,
		id int,
	) (domain.Task, error)

	DeleteTask(
		ctx context.Context,
		id int,
	) error

	PatchTask(
		ctx context.Context,
		id int,
		task domain.Task,
	) (domain.Task, error)

	GetTaskItems(ctx context.Context, taskID int) ([]domain.TaskItem, error)
	GetTaskItem(ctx context.Context, taskID int, itemID int) (domain.TaskItem, error)
	CreateTaskItem(ctx context.Context, item domain.TaskItem) (domain.TaskItem, error)
	PatchTaskItem(ctx context.Context, item domain.TaskItem) (domain.TaskItem, error)
	DeleteTaskItem(ctx context.Context, taskID int, itemID int) error
	CopyTaskItems(ctx context.Context, fromTaskID int, toTaskID int) error
}

func NewTasksService(
	tasksRepository TasksRepository,
	listsRepository ListsRepository,
) *TasksService {
	return &TasksService{
		tasksRepository: tasksRepository,
		listsRepository: listsRepository,
	}
}
