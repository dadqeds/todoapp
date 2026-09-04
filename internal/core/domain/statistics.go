package domain

import "time"

type Statistics struct {
	TasksCreated               int
	TasksComplited             int
	TasksComplitedRate         *float64
	TasksAverageCompletionTime *time.Duration
}

func NewStatistics(
	tasksCreated int,
	tasksComplited int,
	tasksComplitedRate *float64,
	tasksAverageCompletionTime *time.Duration,
) Statistics {
	return Statistics{
		TasksCreated:               tasksCreated,
		TasksComplited:             tasksComplited,
		TasksComplitedRate:         tasksComplitedRate,
		TasksAverageCompletionTime: tasksAverageCompletionTime,
	}
}
