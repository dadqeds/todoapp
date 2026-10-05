package tasks_transport_http

import (
	"net/http"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_request "github.com/dadqeds/todoapp/internal/core/transport/http/request"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
	core_http_types "github.com/dadqeds/todoapp/internal/core/transport/http/types"
)

type TaskItemDTOResponse struct {
	ID        int       `json:"id"         example:"12"`
	Version   int       `json:"version"    example:"1"`
	TaskID    int       `json:"task_id"    example:"5"`
	Title     string    `json:"title"      example:"Молоко"`
	Done      bool      `json:"done"       example:"false"`
	Position  int       `json:"position"   example:"1"`
	CreatedAt time.Time `json:"created_at"`
}

func taskItemDTOFromDomain(i domain.TaskItem) TaskItemDTOResponse {
	return TaskItemDTOResponse{
		ID:        i.ID,
		Version:   i.Version,
		TaskID:    i.TaskID,
		Title:     i.Title,
		Done:      i.Done,
		Position:  i.Position,
		CreatedAt: i.CreatedAt,
	}
}

// GetTaskItems 	godoc
// @Summary 		Чеклист задачи
// @Description 	Пункты задачи по порядку. Доступ — как у задачи
// @Tags 			tasks
// @Security 		TelegramInitData
// @Produce 		json
// @Param			id path int true "ID задачи"
// @Success 		200 {array} TaskItemDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/tasks/{id}/items [get]
func (h *TasksHTTPHandler) GetTaskItems(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	taskID, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get taskID path value")
		return
	}

	items, err := h.tasksService.GetTaskItems(ctx, taskID)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get task items")
		return
	}

	response := make([]TaskItemDTOResponse, len(items))
	for i, item := range items {
		response[i] = taskItemDTOFromDomain(item)
	}

	responseHandler.JSONResponse(response, http.StatusOK)
}

type CreateTaskItemRequest struct {
	Title string `json:"title" validate:"required,min=1,max=200" example:"Молоко"`
}

// CreateTaskItem 	godoc
// @Summary 		Добавить пункт
// @Description 	Пункт встаёт в конец чеклиста. У задачи не больше 100 пунктов (409)
// @Tags 			tasks
// @Security 		TelegramInitData
// @Accept 			json
// @Produce 		json
// @Param			id path int true "ID задачи"
// @Param			request body CreateTaskItemRequest true "Новый пункт"
// @Success 		201 {object} TaskItemDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		409 {object} core_http_response.ErrorResponse "Слишком много пунктов"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/tasks/{id}/items [post]
func (h *TasksHTTPHandler) CreateTaskItem(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	taskID, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get taskID path value")
		return
	}

	var request CreateTaskItemRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	item, err := h.tasksService.CreateTaskItem(ctx, taskID, request.Title)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create task item")
		return
	}

	responseHandler.JSONResponse(taskItemDTOFromDomain(item), http.StatusCreated)
}

type PatchTaskItemRequest struct {
	Title   core_http_types.Nullable[string] `json:"title"   swaggertype:"string"  example:"Молоко 3.2%"`
	Done    core_http_types.Nullable[bool]   `json:"done"    swaggertype:"boolean" example:"true"`
	Version *int                             `json:"version" example:"1"`
}

// PatchTaskItem 	godoc
// @Summary 		Изменить пункт
// @Description 	Текст и отметка. Необязательное поле 'version' защищает от одновременного изменения (409)
// @Tags 			tasks
// @Security 		TelegramInitData
// @Accept 			json
// @Produce 		json
// @Param			id path int true "ID задачи"
// @Param			item_id path int true "ID пункта"
// @Param			request body PatchTaskItemRequest true "Изменения"
// @Success 		200 {object} TaskItemDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		409 {object} core_http_response.ErrorResponse "Conflict"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/tasks/{id}/items/{item_id} [patch]
func (h *TasksHTTPHandler) PatchTaskItem(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	taskID, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get taskID path value")
		return
	}

	itemID, err := core_http_request.GetIntPathValue(r, "item_id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get itemID path value")
		return
	}

	var request PatchTaskItemRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	item, err := h.tasksService.PatchTaskItem(ctx, taskID, itemID, domain.TaskItemPatch{
		Title:           request.Title.ToDomain(),
		Done:            request.Done.ToDomain(),
		ExpectedVersion: request.Version,
	})
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to patch task item")
		return
	}

	responseHandler.JSONResponse(taskItemDTOFromDomain(item), http.StatusOK)
}

// DeleteTaskItem 	godoc
// @Summary 		Удалить пункт
// @Tags 			tasks
// @Security 		TelegramInitData
// @Param			id path int true "ID задачи"
// @Param			item_id path int true "ID пункта"
// @Success 		204 "Пункт удалён"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/tasks/{id}/items/{item_id} [delete]
func (h *TasksHTTPHandler) DeleteTaskItem(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	taskID, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get taskID path value")
		return
	}

	itemID, err := core_http_request.GetIntPathValue(r, "item_id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get itemID path value")
		return
	}

	if err := h.tasksService.DeleteTaskItem(ctx, taskID, itemID); err != nil {
		responseHandler.ErrorResponse(err, "failed to delete task item")
		return
	}

	responseHandler.NoContentResponse()
}
