package domain

import (
	"errors"
	"strings"
	"testing"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func TestTaskItemValidate(t *testing.T) {
	tests := []struct {
		title string
		ok    bool
	}{
		{"Молоко", true},
		{"", false},
		{strings.Repeat("я", TaskItemTitleMaxLen), true},
		{strings.Repeat("я", TaskItemTitleMaxLen+1), false},
	}
	for _, tt := range tests {
		item := NewTaskItemUninitialized(1, tt.title)
		if err := item.Validate(); (err == nil) != tt.ok {
			t.Errorf("title len %d: err = %v, want ok=%v", len([]rune(tt.title)), err, tt.ok)
		}
	}
}

func TestTaskItemApplyPatch(t *testing.T) {
	item := TaskItem{ID: 1, Version: 2, TaskID: 1, Title: "Хлеб"}

	if err := item.ApplyPatch(TaskItemPatch{Done: set(true), ExpectedVersion: ptr(1)}); !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("stale version: err = %v, want ErrConflict", err)
	}
	if err := item.ApplyPatch(TaskItemPatch{Title: setNull[string]()}); !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("null title: err = %v, want ErrInvalidArgument", err)
	}
	if err := item.ApplyPatch(TaskItemPatch{Title: set("")}); !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("empty title: err = %v, want ErrInvalidArgument", err)
	}
	if item.Title != "Хлеб" || item.Done {
		t.Fatalf("failed patch changed item: %+v", item)
	}

	if err := item.ApplyPatch(TaskItemPatch{Done: set(true), Title: set("Батон"), ExpectedVersion: ptr(2)}); err != nil {
		t.Fatal(err)
	}
	if !item.Done || item.Title != "Батон" {
		t.Fatalf("item = %+v", item)
	}
}
