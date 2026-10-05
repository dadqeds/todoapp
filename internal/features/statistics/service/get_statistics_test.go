package statistics_service

import (
	"context"
	"errors"
	"testing"
	"time"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type fakeRepository struct {
	summary   TasksSummary
	called    bool
	gotUserID *int
}

func (f *fakeRepository) GetTasksSummary(_ context.Context, userID *int, _ *time.Time, _ *time.Time) (TasksSummary, error) {
	f.called = true
	f.gotUserID = userID
	return f.summary, nil
}

func asActor(id int, isAdmin bool) context.Context {
	return core_auth.ToContext(context.Background(), core_auth.Actor{User: domain.User{ID: id}, IsAdmin: isAdmin})
}

func TestGetStatistics(t *testing.T) {
	avg := 90 * time.Minute
	repo := &fakeRepository{summary: TasksSummary{Created: 4, Completed: 1, AverageCompletionTime: &avg}}

	stats, err := NewStatisticsService(repo).GetStatistics(asActor(1, true), nil, nil, nil)
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
	stats, err := NewStatisticsService(&fakeRepository{}).GetStatistics(asActor(1, true), nil, nil, nil)
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

	_, err := NewStatisticsService(repo).GetStatistics(asActor(1, true), nil, &from, &to)
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}
	if repo.called {
		t.Fatal("repository must not be called for invalid range")
	}
}

func TestGetStatisticsScope(t *testing.T) {
	other := 2

	t.Run("user sees only own statistics", func(t *testing.T) {
		repo := &fakeRepository{}

		if _, err := NewStatisticsService(repo).GetStatistics(asActor(7, false), &other, nil, nil); err != nil {
			t.Fatal(err)
		}
		if repo.gotUserID == nil || *repo.gotUserID != 7 {
			t.Fatalf("user_id = %v, want 7", repo.gotUserID)
		}
	})

	t.Run("admin may filter by any user", func(t *testing.T) {
		repo := &fakeRepository{}

		if _, err := NewStatisticsService(repo).GetStatistics(asActor(7, true), &other, nil, nil); err != nil {
			t.Fatal(err)
		}
		if repo.gotUserID == nil || *repo.gotUserID != other {
			t.Fatalf("user_id = %v, want %d", repo.gotUserID, other)
		}
	})

	t.Run("no user in context", func(t *testing.T) {
		_, err := NewStatisticsService(&fakeRepository{}).GetStatistics(context.Background(), nil, nil, nil)
		if !errors.Is(err, core_errors.ErrUnauthenticated) {
			t.Fatalf("err = %v, want ErrUnauthenticated", err)
		}
	})
}
