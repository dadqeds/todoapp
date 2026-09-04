package statistics_service

import (
	"context"
	"fmt"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func(s *StatisticsService) GetStatistics(
	ctx context.Context,
	userID *int,
	from *time.Time,
	to *time.Time,
)(domain.Statistics, error){
	if from !=nil && to !=nil{
		if to.Before(*from) || to.Equal(*from){
			return domain.Statistics{}, fmt.Errorf(
				"`to` must be after `from`: %w",
				core_errors.ErrInvalidArgument,
			)
		}
	}

	tasks, err :=s.statisticsRepository.GetTasks(ctx,userID,from,to)
	if err != nil{
		return domain.Statistics{}, fmt.Errorf("get tasks from repository: %w", err)
	}

	statistics := calcStatistics(tasks)


	return statistics, nil
}

func calcStatistics(tasks []domain.Task) domain.Statistics{
	if len(tasks) == 0{
		return domain.Statistics{
			TasksCreated: 0,
			TasksComplited: 0,
			TasksComplitedRate: nil,
			TasksAverageCompletionTime: nil,
		}
	}

	tasksCreated := len(tasks)
	var totalCompletionDuration time.Duration
	tasksComplited := 0
	for _, task := range tasks{
		if task.Completed{
			tasksComplited++
		}
		
		completionDuration :=task.ComplitedDuration()
		if completionDuration != nil{
			totalCompletionDuration += *completionDuration
		}
	}

	tasksComplitedRate := float64(tasksComplited) / float64(tasksCreated) * 100

	var tasksAverageCompletionTime *time.Duration
	if tasksComplited > 0 && totalCompletionDuration != 0{
		avg := totalCompletionDuration / time.Duration(tasksComplited)

		tasksAverageCompletionTime = &avg
	}

	return domain.Statistics{
		TasksCreated: tasksCreated,
		TasksComplited: tasksComplited,
		TasksComplitedRate: &tasksComplitedRate,
		TasksAverageCompletionTime: tasksAverageCompletionTime,
	}
}