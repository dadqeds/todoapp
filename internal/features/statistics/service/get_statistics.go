package statistics_service

import (
	"context"
	"fmt"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func (s *StatisticsService) GetStatistics(
	ctx context.Context,
	userID *int,
	from *time.Time,
	to *time.Time,
) (domain.Statistics, error) {
	if from != nil && to != nil {
		if !to.After(*from) {
			return domain.Statistics{}, fmt.Errorf(
				"`to` must be after `from`: %w",
				core_errors.ErrInvalidArgument,
			)
		}
	}

	summary, err := s.statisticsRepository.GetTasksSummary(ctx, userID, from, to)
	if err != nil {
		return domain.Statistics{}, fmt.Errorf("get tasks summary from repository: %w", err)
	}

	return calcStatistics(summary), nil
}

func calcStatistics(summary TasksSummary) domain.Statistics {
	if summary.Created == 0 {
		return domain.NewStatistics(0, 0, nil, nil)
	}

	completedRate := float64(summary.Completed) / float64(summary.Created) * 100

	return domain.NewStatistics(
		summary.Created,
		summary.Completed,
		&completedRate,
		summary.AverageCompletionTime,
	)
}
