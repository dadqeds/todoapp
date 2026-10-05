package statistics_transport_http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_request "github.com/dadqeds/todoapp/internal/core/transport/http/request"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

type GetStatisticsResponse struct {
	TasksCreated               int      `json:"tasks_created"                        example:"50"`
	TasksCompleted             int      `json:"tasks_completed"                      example:"10"`
	TasksCompletedRate         *float64 `json:"tasks_completed_rate"                 example:"20"`
	TasksAverageCompletionTime *string  `json:"tasks_average_completion_time"        example:"1m30s"`
	// Секунды — чтобы фронт мог показать «1 день 4 ч», а не разбирать строку Go.
	TasksAverageCompletionSeconds *int64 `json:"tasks_average_completion_seconds" example:"90"`

	TasksCompletedWithDue int      `json:"tasks_completed_with_due" example:"6"`
	TasksCompletedOnTime  int      `json:"tasks_completed_on_time"  example:"5"`
	TasksOnTimeRate       *float64 `json:"tasks_on_time_rate"       example:"83.3"`

	Lists []ListStatisticsResponse `json:"lists"`
}

type ListStatisticsResponse struct {
	ListID         int    `json:"list_id"         example:"3"`
	Title          string `json:"title"           example:"Дом"`
	Color          string `json:"color"           example:"green"`
	TasksCreated   int    `json:"tasks_created"   example:"9"`
	TasksCompleted int    `json:"tasks_completed" example:"7"`
}

// GetStatistics	godoc
// @Summary Получение статистики
// @Description Получение статистики по задачам с опциональной фильтрацией по user_id и/или временному промежутку
// @Tags	statistics
// @Security 		TelegramInitData
// @Produce	json
// @Param 	user_id query int false "Фильтрация статистики по конкретному пользователю"
// @Param 	from query string false "Начало промежутка рассмотрения статистики (включительно), формат: YYYY-MM-DD"
// @Param 	to query string false "Конец промежутка рассмотрения статистики (не включительно), формат: YYYY-MM-DD"
// @Success 200 {object} GetStatisticsResponse "Успешное получение"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router /statistics [get]
func (h *StatisticsHTTPHandler) GetStatistics(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	userID, from, to, err := getUserIdFromToQueryParams(r)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get userID/from/to query params",
		)
		return
	}

	statistics, err := h.statisticsService.GetStatistics(ctx, userID, from, to)
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get statistics ",
		)
		return
	}

	response := toDTOFromDomain(statistics)

	responseHandler.JSONResponse(response, http.StatusOK)
}

func toDTOFromDomain(statistics domain.Statistics) GetStatisticsResponse {
	var avgTime *string
	if statistics.TasksAverageCompletionTime != nil {
		duration := statistics.TasksAverageCompletionTime.Round(time.Second).String()
		avgTime = &duration
	}
	var avgSeconds *int64
	if statistics.TasksAverageCompletionTime != nil {
		sec := int64(statistics.TasksAverageCompletionTime.Round(time.Second) / time.Second)
		avgSeconds = &sec
	}

	lists := make([]ListStatisticsResponse, len(statistics.Lists))
	for i, l := range statistics.Lists {
		lists[i] = ListStatisticsResponse{
			ListID:         l.ListID,
			Title:          l.Title,
			Color:          l.Color,
			TasksCreated:   l.TasksCreated,
			TasksCompleted: l.TasksCompleted,
		}
	}

	return GetStatisticsResponse{
		TasksAverageCompletionSeconds: avgSeconds,
		TasksCompletedWithDue:         statistics.TasksCompletedWithDue,
		TasksCompletedOnTime:          statistics.TasksCompletedOnTime,
		TasksOnTimeRate:               statistics.TasksOnTimeRate,
		Lists:                         lists,
		TasksCreated:                  statistics.TasksCreated,
		TasksCompleted:                statistics.TasksCompleted,
		TasksCompletedRate:            statistics.TasksCompletedRate,
		TasksAverageCompletionTime:    avgTime,
	}
}

func getUserIdFromToQueryParams(r *http.Request) (*int, *time.Time, *time.Time, error) {
	const (
		userIDQueryParamKey = "user_id"
		fromQueryParamKey   = "from"
		toQueryParamKey     = "to"
	)

	userID, err := core_http_request.GetIntQueryParam(r, userIDQueryParamKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get 'user_id' query param: %w", err)
	}

	from, err := core_http_request.GetDateQueryParam(r, fromQueryParamKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get 'from' query param: %w", err)
	}

	to, err := core_http_request.GetDateQueryParam(r, toQueryParamKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get 'to' query param: %w", err)
	}

	return userID, from, to, nil
}
