package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func ptr[T any](v T) *T { return &v }

func set[T any](v T) Nullable[T] { return Nullable[T]{Value: &v, Set: true} }

func setNull[T any]() Nullable[T] { return Nullable[T]{Set: true} }

func TestTaskCompletionDuration(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		task Task
		want *time.Duration
	}{
		{
			name: "not completed",
			task: Task{CreatedAt: createdAt},
			want: nil,
		},
		{
			name: "completed without completed_at",
			task: Task{CreatedAt: createdAt, Completed: true},
			want: nil,
		},
		{
			name: "completed",
			task: Task{
				CreatedAt:   createdAt,
				Completed:   true,
				CompletedAt: ptr(createdAt.Add(90 * time.Minute)),
			},
			want: ptr(90 * time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.CompletionDuration()

			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("got %v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Fatalf("got nil, want %v", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Fatalf("got %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestTaskValidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		task    Task
		wantErr bool
	}{
		{"valid", Task{Title: "a", CreatedAt: now}, false},
		{"empty title", Task{Title: "", CreatedAt: now}, true},
		{"title 100 runes", Task{Title: strings.Repeat("я", 100), CreatedAt: now}, false},
		{"title 101 runes", Task{Title: strings.Repeat("я", 101), CreatedAt: now}, true},
		{"empty description", Task{Title: "a", Description: ptr(""), CreatedAt: now}, true},
		{"completed without completed_at", Task{Title: "a", Completed: true, CreatedAt: now}, true},
		{"completed_at before created_at", Task{Title: "a", Completed: true, CreatedAt: now, CompletedAt: ptr(now.Add(-time.Second))}, true},
		{"not completed with completed_at", Task{Title: "a", CreatedAt: now, CompletedAt: ptr(now)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, core_errors.ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestTaskApplyPatch(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour)
	completedAt := createdAt.Add(10 * time.Minute)

	t.Run("complete open task sets completed_at", func(t *testing.T) {
		task := Task{Title: "a", CreatedAt: createdAt}

		if err := task.ApplyPatch(TaskPatch{Completed: set(true)}); err != nil {
			t.Fatal(err)
		}
		if !task.Completed || task.CompletedAt == nil {
			t.Fatalf("task not completed: %+v", task)
		}
	})

	t.Run("completing already completed task keeps completed_at", func(t *testing.T) {
		task := Task{Title: "a", CreatedAt: createdAt, Completed: true, CompletedAt: ptr(completedAt)}

		if err := task.ApplyPatch(TaskPatch{Completed: set(true)}); err != nil {
			t.Fatal(err)
		}
		if task.CompletedAt == nil || !task.CompletedAt.Equal(completedAt) {
			t.Fatalf("completed_at changed: got %v, want %v", task.CompletedAt, completedAt)
		}
	})

	t.Run("reopen clears completed_at", func(t *testing.T) {
		task := Task{Title: "a", CreatedAt: createdAt, Completed: true, CompletedAt: ptr(completedAt)}

		if err := task.ApplyPatch(TaskPatch{Completed: set(false)}); err != nil {
			t.Fatal(err)
		}
		if task.Completed || task.CompletedAt != nil {
			t.Fatalf("task not reopened: %+v", task)
		}
	})

	t.Run("null description clears it", func(t *testing.T) {
		task := Task{Title: "a", Description: ptr("d"), CreatedAt: createdAt}

		if err := task.ApplyPatch(TaskPatch{Description: setNull[string]()}); err != nil {
			t.Fatal(err)
		}
		if task.Description != nil {
			t.Fatalf("description = %q, want nil", *task.Description)
		}
	})

	t.Run("unset fields stay untouched", func(t *testing.T) {
		task := Task{Title: "a", Description: ptr("d"), CreatedAt: createdAt}

		if err := task.ApplyPatch(TaskPatch{Title: set("b")}); err != nil {
			t.Fatal(err)
		}
		if task.Title != "b" || task.Description == nil || *task.Description != "d" {
			t.Fatalf("unexpected task: %+v", task)
		}
	})

	invalid := []struct {
		name  string
		patch TaskPatch
	}{
		{"null title", TaskPatch{Title: setNull[string]()}},
		{"null completed", TaskPatch{Completed: setNull[bool]()}},
		{"too long title", TaskPatch{Title: set(strings.Repeat("a", 101))}},
	}
	for _, tt := range invalid {
		t.Run(tt.name+" is rejected and task is unchanged", func(t *testing.T) {
			task := Task{Title: "a", CreatedAt: createdAt}
			before := task

			err := task.ApplyPatch(tt.patch)
			if !errors.Is(err, core_errors.ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
			if task != before {
				t.Fatalf("task changed on failed patch: %+v", task)
			}
		})
	}
}

func TestTaskApplyPatchExpectedVersion(t *testing.T) {
	task := Task{Version: 3, Title: "a", CreatedAt: time.Now()}

	if err := task.ApplyPatch(TaskPatch{Title: set("b"), ExpectedVersion: ptr(2)}); !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("stale version: err = %v, want ErrConflict", err)
	}
	if task.Title != "a" {
		t.Fatalf("task changed on conflict: %+v", task)
	}

	if err := task.ApplyPatch(TaskPatch{Title: set("b"), ExpectedVersion: ptr(3)}); err != nil {
		t.Fatalf("matching version: %v", err)
	}
}
