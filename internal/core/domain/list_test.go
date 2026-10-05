package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func TestListValidate(t *testing.T) {
	tests := []struct {
		name    string
		list    List
		wantErr bool
	}{
		{"valid", List{Title: "Дом", Color: "green"}, false},
		{"empty title", List{Title: "", Color: "green"}, true},
		{"long title", List{Title: strings.Repeat("я", 51), Color: "green"}, true},
		{"unknown color", List{Title: "Дом", Color: "#ff0000"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.list.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListApplyPatch(t *testing.T) {
	list := List{Version: 2, Title: "Дом", Color: "green"}

	if err := list.ApplyPatch(ListPatch{Color: set("pink"), ExpectedVersion: ptr(1)}); !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("stale version: err = %v", err)
	}
	if err := list.ApplyPatch(ListPatch{Color: set("rainbow")}); !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("bad color: err = %v", err)
	}
	if list.Color != "green" {
		t.Fatalf("list changed on failed patch: %+v", list)
	}
	if err := list.ApplyPatch(ListPatch{Title: set("Дача"), Color: set("amber")}); err != nil {
		t.Fatal(err)
	}
	if list.Title != "Дача" || list.Color != "amber" {
		t.Fatalf("patch not applied: %+v", list)
	}
}

func TestTaskDue(t *testing.T) {
	now := time.Now()
	due := now.Add(time.Hour)

	if err := (&Task{Title: "a", CreatedAt: now, DueAllDay: true}).Validate(); !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("all-day without due: err = %v", err)
	}

	task := Task{Title: "a", CreatedAt: now, DueAt: &due, DueAllDay: true}
	if err := task.ApplyPatch(TaskPatch{DueAt: setNull[time.Time]()}); err != nil {
		t.Fatal(err)
	}
	if task.DueAt != nil || task.DueAllDay {
		t.Fatalf("clearing due must reset all-day: %+v", task)
	}

	tests := []struct {
		name string
		task Task
		want bool
	}{
		{"open", Task{DueAt: &due}, false},
		{"completed before due", Task{Completed: true, CompletedAt: &now, DueAt: &due}, true},
		{"completed after due", Task{Completed: true, CompletedAt: ptr(due.Add(time.Minute)), DueAt: &due}, false},
		{"completed without due", Task{Completed: true, CompletedAt: &now}, false},
	}
	for _, tt := range tests {
		if got := tt.task.IsCompletedOnTime(); got != tt.want {
			t.Errorf("%s: IsCompletedOnTime = %v, want %v", tt.name, got, tt.want)
		}
	}
}
