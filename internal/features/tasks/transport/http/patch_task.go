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

type PatchTaskRequest struct {
	Title       core_http_types.Nullable[string] `json:"title"       swaggertype:"string"  example:"Купить молоко"`
	Description core_http_types.Nullable[string] `json:"description" swaggertype:"string"  example:"2 литра"`
	Completed   core_http_types.Nullable[bool]   `json:"completed"   swaggertype:"boolean" example:"true"`
	ListID      core_http_types.Nullable[int]    `json:"list_id"     swaggertype:"integer" example:"3"`
	// null снимает срок
	DueAt     core_http_types.Nullable[time.Time] `json:"due_at"      swaggertype:"string"  example:"2026-10-06T10:00:00+03:00"`
	DueAllDay core_http_types.Nullable[bool]      `json:"due_all_day" swaggertype:"boolean" example:"false"`
	// null выключает повтор
	Repeat core_http_types.Nullable[RepeatDTO] `json:"repeat"`
	// null выключает напоминание
	RemindBeforeMinutes core_http_types.Nullable[int] `json:"remind_before_minutes" swaggertype:"integer" example:"60"`
	Version             *int                          `json:"version"     example:"3"`
}

type PatchTaskResponse TaskDTOResponse

// PatchTask 		godoc
// @Summary 		Обновить задачу
// @Description 	Обновляет информацию об уже существующей в системе задаче
// @Description 	### Логика обновления полей (Three-state logic):
// @Description 	1. **Поле не передано**: 'description' игнорируется, значение в БД не меняется
// @Description 	2. **Явно передано значение**: '"description": "Утром в 06:30 выйти на прогулку с Бобиком"'
// @Description 	3. **Явно передан null**: '"description": null' - очищает поле в БД (set to NULL)
// @Description 	Ограничения: 'title' и 'completed' не могут быть выставлены как null
// @Description 	Необязательное поле 'version' — версия, которую видел клиент; при расхождении с текущей вернётся 409
// @Description 	Выполнение повторяющейся задачи создаёт следующую с новым сроком, у выполненной повтор снимается
// @Tags 			tasks
// @Security 		TelegramInitData
// @Accept 			json
// @Produce 		json
// @Param 			id path int true "ID изменяемой задачи"
// @Param 			request body PatchTaskRequest true "PatchTask тело запроса"
// @Success 		200 {object} PatchTaskResponse "Успешно изменённая задача"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		409 {object} core_http_response.ErrorResponse "Conflict"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/tasks/{id} [patch]
func (h *TasksHTTPHandler) PatchTask(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	taskID, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get taskID in path value",
		)
		return
	}

	var request PatchTaskRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to decode and validate HTTP request",
		)

		return
	}

	taskPatch := taskPatchFromRequest(request)

	taskDomain, err := h.tasksService.PatchTask(ctx, taskID, taskPatch)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to patch task",
		)

		return
	}

	response := PatchTaskResponse(taskDTOFromDomain(taskDomain))
	responseHandler.JSONResponse(response, http.StatusOK)
}

func taskPatchFromRequest(request PatchTaskRequest) domain.TaskPatch {
	return domain.TaskPatch{
		Title:       request.Title.ToDomain(),
		Description: request.Description.ToDomain(),
		Completed:   request.Completed.ToDomain(),
		ListID:      request.ListID.ToDomain(),
		DueAt:       request.DueAt.ToDomain(),
		DueAllDay:   request.DueAllDay.ToDomain(),
		Repeat:      domain.Nullable[domain.Recurrence]{Value: request.Repeat.Value.toDomain(), Set: request.Repeat.Set},

		RemindBeforeMinutes: request.RemindBeforeMinutes.ToDomain(),
		ExpectedVersion:     request.Version,
	}
}
