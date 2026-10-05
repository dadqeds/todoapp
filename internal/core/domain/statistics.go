package domain

import "time"

type Statistics struct {
	TasksCreated               int
	TasksCompleted             int
	TasksCompletedRate         *float64
	TasksAverageCompletionTime *time.Duration

	// Из выполненных задач со сроком — сколько выполнено не позже срока.
	TasksCompletedWithDue int
	TasksCompletedOnTime  int
	TasksOnTimeRate       *float64

	Lists []ListStatistics
}

// ListStatistics — счётчики задач одного списка за период.
type ListStatistics struct {
	ListID         int
	Title          string
	Color          string
	TasksCreated   int
	TasksCompleted int
}
