package statistics_service

import (
	"context"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

type StatisticsService struct {
	statisticsRepository StatisticsRepository
}

// TasksSummary — агрегаты по задачам, посчитанные хранилищем.
type TasksSummary struct {
	Created               int
	Completed             int
	AverageCompletionTime *time.Duration

	CompletedWithDue int
	CompletedOnTime  int

	Lists []domain.ListStatistics
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
