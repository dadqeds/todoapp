package tasks_service

import (
	"testing"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func newServiceWithChanges(repo *fakeTasksRepository) (*TasksService, *fakeChanges) {
	changes := &fakeChanges{}
	return NewTasksService(repo, fakeListsRepository{}, changes), changes
}

func TestCreateTaskEnqueuesChange(t *testing.T) {
	s, changes := newServiceWithChanges(newRepo())

	// Участник 30 добавляет задачу в общий список 200.
	if _, err := s.CreateTask(asActor(30, false), domain.NewTaskUninitialized("Купить молоко", nil, 0, 200, nil, false)); err != nil {
		t.Fatal(err)
	}

	want := domain.ListChange{ListID: 200, ActorUserID: 30, Kind: domain.ListChangeTaskAdded, TaskTitle: "Купить молоко"}
	if len(changes.changes) != 1 || changes.changes[0] != want {
		t.Fatalf("changes = %+v, want [%+v]", changes.changes, want)
	}
}

func TestPatchTaskEnqueuesOnlyCompletion(t *testing.T) {
	repo := newRepo()
	s, changes := newServiceWithChanges(repo)
	member := asActor(30, false)
	done, undone, title := true, false, "Вынести мусор"

	if _, err := s.PatchTask(member, 2, domain.TaskPatch{Title: domain.Nullable[string]{Value: &title, Set: true}}); err != nil {
		t.Fatal(err)
	}
	if len(changes.changes) != 0 {
		t.Fatalf("title change enqueued: %+v", changes.changes)
	}

	if _, err := s.PatchTask(member, 2, domain.TaskPatch{Completed: domain.Nullable[bool]{Value: &done, Set: true}}); err != nil {
		t.Fatal(err)
	}
	want := domain.ListChange{ListID: 200, ActorUserID: 30, Kind: domain.ListChangeTaskCompleted, TaskTitle: "foreign"}
	if len(changes.changes) != 1 || changes.changes[0] != want {
		t.Fatalf("changes = %+v, want [%+v]", changes.changes, want)
	}

	// Вернуть в работу — не событие.
	repo.tasks[2] = repo.patched
	if _, err := s.PatchTask(member, 2, domain.TaskPatch{Completed: domain.Nullable[bool]{Value: &undone, Set: true}}); err != nil {
		t.Fatal(err)
	}
	if len(changes.changes) != 1 {
		t.Fatalf("reopening enqueued: %+v", changes.changes)
	}
}

func TestChecklistDoesNotEnqueueChanges(t *testing.T) {
	repo := newRepo()
	s, changes := newServiceWithChanges(repo)
	member := asActor(30, false)
	done := true

	item, err := s.CreateTaskItem(member, 2, "Пакеты")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PatchTaskItem(member, 2, item.ID, domain.TaskItemPatch{Done: domain.Nullable[bool]{Value: &done, Set: true}}); err != nil {
		t.Fatal(err)
	}
	if len(changes.changes) != 0 {
		t.Fatalf("checklist enqueued changes: %+v", changes.changes)
	}
}
