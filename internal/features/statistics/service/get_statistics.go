package statistics_service

import (
	"context"
	"fmt"
	"time"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func (s *StatisticsService) GetStatistics(
	ctx context.Context,
	userID *int,
	from *time.Time,
	to *time.Time,
) (domain.Statistics, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.Statistics{}, err
	}

	// Обычный пользователь видит статистику только по своим задачам.
	if !actor.IsAdmin {
		userID = &actor.User.ID
	}

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

func percent(part, total int) *float64 {
	if total == 0 {
		return nil
	}
	rate := float64(part) / float64(total) * 100
	return &rate
}

func calcStatistics(summary TasksSummary) domain.Statistics {
	return domain.Statistics{
		TasksCreated:               summary.Created,
		TasksCompleted:             summary.Completed,
		TasksCompletedRate:         percent(summary.Completed, summary.Created),
		TasksAverageCompletionTime: summary.AverageCompletionTime,
		TasksCompletedWithDue:      summary.CompletedWithDue,
		TasksCompletedOnTime:       summary.CompletedOnTime,
		TasksOnTimeRate:            percent(summary.CompletedOnTime, summary.CompletedWithDue),
		Lists:                      summary.Lists,
	}
}
