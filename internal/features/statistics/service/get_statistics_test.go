package statistics_service

import (
	"context"
	"errors"
	"testing"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type fakeRepository struct {
	summary TasksSummary
	called  bool
}

func (f *fakeRepository) GetTasksSummary(context.Context, *int, *time.Time, *time.Time) (TasksSummary, error) {
	f.called = true
	return f.summary, nil
}

func TestGetStatistics(t *testing.T) {
	avg := 90 * time.Minute
	repo := &fakeRepository{summary: TasksSummary{Created: 4, Completed: 1, AverageCompletionTime: &avg}}

	stats, err := NewStatisticsService(repo).GetStatistics(context.Background(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if stats.TasksCreated != 4 || stats.TasksCompleted != 1 {
		t.Fatalf("unexpected counters: %+v", stats)
	}
	if stats.TasksCompletedRate == nil || *stats.TasksCompletedRate != 25 {
		t.Fatalf("rate = %v, want 25", stats.TasksCompletedRate)
	}
	if stats.TasksAverageCompletionTime == nil || *stats.TasksAverageCompletionTime != avg {
		t.Fatalf("avg = %v, want %v", stats.TasksAverageCompletionTime, avg)
	}
}

func TestGetStatisticsEmpty(t *testing.T) {
	stats, err := NewStatisticsService(&fakeRepository{}).GetStatistics(context.Background(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if stats.TasksCreated != 0 || stats.TasksCompletedRate != nil || stats.TasksAverageCompletionTime != nil {
		t.Fatalf("unexpected stats for empty summary: %+v", stats)
	}
}

func TestGetStatisticsInvalidRange(t *testing.T) {
	from := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	to := from
	repo := &fakeRepository{}

	_, err := NewStatisticsService(repo).GetStatistics(context.Background(), nil, &from, &to)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}
	if repo.called {
		t.Fatal("repository must not be called for invalid range")
	}
}
