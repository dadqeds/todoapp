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
	TasksComplited             int      `json:"tasks_complited"                      example:"10"`
	TasksComplitedRate         *float64 `json:"tasks_complited_rate"                 example:"20"`
	TasksAverageCompletionTime *string  `json:"tasks_average_completion_time"        example:"1m30s"`
}

// GetStatistics	godoc
// @Summary Получение статистики
// @Description Получение статистики по задачам с опциональной фильтрацией по user_id и/или временному промежутку
// @Tags	statistics
// @Produce	json
// @Param 	user_id query int false "Фильтрация статистики по конкретному пользователю"
// @Param 	from query string false "Начало промежутка рассмотрения статистики (включительно), формат: YYYY-MM-DD"
// @Param 	to query string false "Конец промежутся рассмотрения статистика (не включительно), формат: YYYY-MM-DD"
// @Success 200 {object} GetStatisticsResponse "Успешное получение"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
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
		Duration := statistics.TasksAverageCompletionTime.String()
		avgTime = &Duration
	}
	return GetStatisticsResponse{
		TasksCreated:               statistics.TasksCreated,
		TasksComplited:             statistics.TasksComplited,
		TasksComplitedRate:         statistics.TasksComplitedRate,
		TasksAverageCompletionTime: avgTime,
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
