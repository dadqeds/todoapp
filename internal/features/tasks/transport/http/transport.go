package tasks_transport_http

import (
	"context"
	"net/http"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_http_server "github.com/dadqeds/todoapp/internal/core/transport/http/server"
)

type TasksHTTPHandler struct {
	tasksService TasksService
}

type TasksService interface {
	CreateTask(
		ctx context.Context,
		task domain.Task,
	) (domain.Task, error)

	GetTasks(
		ctx context.Context,
		filter domain.TaskFilter,
		limit *int,
		offset *int,
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
		patch domain.TaskPatch,
	) (domain.Task, error)

	GetTaskItems(ctx context.Context, taskID int) ([]domain.TaskItem, error)
	CreateTaskItem(ctx context.Context, taskID int, title string) (domain.TaskItem, error)
	PatchTaskItem(ctx context.Context, taskID int, itemID int, patch domain.TaskItemPatch) (domain.TaskItem, error)
	DeleteTaskItem(ctx context.Context, taskID int, itemID int) error
}

func NewTasksHTTPHandler(
	tasksService TasksService,
) *TasksHTTPHandler {
	return &TasksHTTPHandler{
		tasksService: tasksService,
	}
}

func (h *TasksHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodPost,
			Path:    "/tasks",
			Handler: h.CreateTask,
		},
		{
			Method:  http.MethodGet,
			Path:    "/tasks",
			Handler: h.GetTasks,
		},
		{
			Method:  http.MethodGet,
			Path:    "/tasks/{id}",
			Handler: h.GetTask,
		},
		{
			Method:  http.MethodDelete,
			Path:    "/tasks/{id}",
			Handler: h.DeleteTask,
		},
		{
			Method:  http.MethodPatch,
			Path:    "/tasks/{id}",
			Handler: h.PatchTask,
		},
		{Method: http.MethodGet, Path: "/tasks/{id}/items", Handler: h.GetTaskItems},
		{Method: http.MethodPost, Path: "/tasks/{id}/items", Handler: h.CreateTaskItem},
		{Method: http.MethodPatch, Path: "/tasks/{id}/items/{item_id}", Handler: h.PatchTaskItem},
		{Method: http.MethodDelete, Path: "/tasks/{id}/items/{item_id}", Handler: h.DeleteTaskItem},
	}
}
