package statistics_service

import (
	"context"
	"time"
)

type StatisticsService struct {
	statisticsRepository StatisticsRepository
}

// TasksSummary — агрегаты по задачам, посчитанные хранилищем.
type TasksSummary struct {
	Created               int
	Completed             int
	AverageCompletionTime *time.Duration
}

type StatisticsRepository interface {
	GetTasksSummary(
		ctx context.Context,
		userID *int,
		from *time.Time,
		to *time.Time,
	) (TasksSummary, error)
}

func NewStatisticsService(
	statisticsRepository StatisticsRepository,
) *StatisticsService {
	return &StatisticsService{
		statisticsRepository: statisticsRepository,
	}
}
