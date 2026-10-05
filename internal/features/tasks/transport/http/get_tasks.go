package tasks_transport_http

import (
	"fmt"
	"net/http"

	"github.com/dadqeds/todoapp/internal/core/domain"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_request "github.com/dadqeds/todoapp/internal/core/transport/http/request"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

type GetTasksResponse []TaskDTOResponse

// GetTasks 		godoc
// @Summary 		Список задач
// @Description 	Сначала невыполненные, по ближайшему сроку; без срока — в конце
// @Tags 			tasks
// @Security 		TelegramInitData
// @Produce 		json
// @Param 			user_id query int false "Фильтрация задач по ID автора (только для администратора)"
// @Param 			list_id query int false "Задачи одного списка"
// @Param 			limit query int false "Размер страницы с задачами (по умолчанию 50, максимум 500)"
// @Param			offset query int false "Смещение страницы с задачами"
// @Success 		200 {object} GetTasksResponse "Успешное получение списка задач"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/tasks [get]
func (h *TasksHTTPHandler) GetTasks(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	filter, limit, offset, err := getTasksQueryParams(r)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get UserID/Limit/Offset query params",
		)

		return
	}

	tasksDomains, err := h.tasksService.GetTasks(ctx, filter, limit, offset)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get tasks",
		)

		return
	}

	response := GetTasksResponse(taskDTOFromDomains(tasksDomains))

	responseHandler.JSONResponse(response, http.StatusOK)
}

func getTasksQueryParams(r *http.Request) (domain.TaskFilter, *int, *int, error) {
	var filter domain.TaskFilter

	userID, err := core_http_request.GetIntQueryParam(r, "user_id")
	if err != nil {
		return filter, nil, nil, fmt.Errorf("get 'user_id' query param: %w", err)
	}
	filter.AuthorUserID = userID

	listID, err := core_http_request.GetIntQueryParam(r, "list_id")
	if err != nil {
		return filter, nil, nil, fmt.Errorf("get 'list_id' query param: %w", err)
	}
	filter.ListID = listID

	limit, err := core_http_request.GetIntQueryParam(r, "limit")
	if err != nil {
		return filter, nil, nil, fmt.Errorf("get 'limit' query param: %w", err)
	}

	offset, err := core_http_request.GetIntQueryParam(r, "offset")
	if err != nil {
		return filter, nil, nil, fmt.Errorf("get 'offset' query param: %w", err)
	}

	return filter, limit, offset, nil
}
