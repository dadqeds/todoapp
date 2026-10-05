package tasks_transport_http

import (
	"net/http"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_request "github.com/dadqeds/todoapp/internal/core/transport/http/request"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

type CreateTaskRequest struct {
	Title       string  `json:"title"       validate:"required,min=1,max=100"   example:"Купить молоко"`
	Description *string `json:"description" validate:"omitempty,min=1,max=1000" example:"2 литра, 3.2%"`
	// Учитывается только для администратора, остальным автором ставится текущий пользователь.
	AuthorUserID int `json:"author_user_id" validate:"omitempty,min=1" example:"1"`
	// Не указан — список по умолчанию автора.
	ListID int `json:"list_id" validate:"omitempty,min=1" example:"3"`
	// Срок с часовым поясом. Для срока «на весь день» — конец дня по времени пользователя.
	DueAt     *time.Time `json:"due_at"      example:"2026-10-05T18:00:00+03:00"`
	DueAllDay bool       `json:"due_all_day" example:"false"`
	// Повтор требует срока.
	Repeat *RepeatDTO `json:"repeat"`
	// 0 — в срок, 15, 60, 1440 — за день; требует срока.
	RemindBeforeMinutes *int `json:"remind_before_minutes" example:"15"`
}

type CreateTaskResponse TaskDTOResponse

// CreateTask 		godoc
// @Summary 		Создать задачу
// @Description 	Создать новую задачу в системе
// @Tags 			tasks
// @Security 		TelegramInitData
// @Accept 			json
// @Produce 		json
// @Param			request body CreateTaskRequest true "CreateTask тело запроса"
// @Success 201		{object} CreateTaskResponse "Успешно созданная задача"
// @Failure 400 	{object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 404 	{object} core_http_response.ErrorResponse "Автор не найден"
// @Failure 500 	{object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/tasks [post]
func (h *TasksHTTPHandler) CreateTask(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	var request CreateTaskRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to decode and validate HTTP request",
		)

		return
	}

	taskDomain := domain.NewTaskUninitialized(
		request.Title,
		request.Description,
		request.AuthorUserID,
		request.ListID,
		request.DueAt,
		request.DueAllDay,
	)

	taskDomain.Repeat = request.Repeat.toDomain()
	taskDomain.RemindBeforeMinutes = request.RemindBeforeMinutes

	taskDomain, err := h.tasksService.CreateTask(ctx, taskDomain)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to create task",
		)
		return
	}

	response := CreateTaskResponse(taskDTOFromDomain(taskDomain))

	responseHandler.JSONResponse(response, http.StatusCreated)
}
