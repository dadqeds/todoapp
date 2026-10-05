package tasks_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func (f *fakeTasksRepository) GetTaskItems(_ context.Context, taskID int) ([]domain.TaskItem, error) {
	var out []domain.TaskItem
	for _, i := range f.items {
		if i.TaskID == taskID {
			out = append(out, i)
		}
	}
	return out, nil
}

func (f *fakeTasksRepository) GetTaskItem(_ context.Context, taskID, itemID int) (domain.TaskItem, error) {
	i, ok := f.items[itemID]
	if !ok || i.TaskID != taskID {
		return domain.TaskItem{}, core_errors.ErrNotFound
	}
	return i, nil
}

func (f *fakeTasksRepository) CreateTaskItem(_ context.Context, item domain.TaskItem) (domain.TaskItem, error) {
	if f.items == nil {
		f.items = map[int]domain.TaskItem{}
	}
	item.ID = len(f.items) + 1
	item.Version = 1
	f.items[item.ID] = item
	return item, nil
}

func (f *fakeTasksRepository) PatchTaskItem(_ context.Context, item domain.TaskItem) (domain.TaskItem, error) {
	item.Version++
	f.items[item.ID] = item
	return item, nil
}

func (f *fakeTasksRepository) DeleteTaskItem(_ context.Context, taskID, itemID int) error {
	if _, err := f.GetTaskItem(context.Background(), taskID, itemID); err != nil {
		return err
	}
	delete(f.items, itemID)
	return nil
}

func (f *fakeTasksRepository) CopyTaskItems(_ context.Context, from, to int) error {
	f.copied = append(f.copied, [2]int{from, to})
	return nil
}

func TestTaskItemsFollowTaskAccess(t *testing.T) {
	repo := newRepo()
	repo.items = map[int]domain.TaskItem{
		1: {ID: 1, Version: 1, TaskID: 2, Title: "чужой"},
	}
	s := newService(repo)
	stranger := asActor(10, false)
	done := true

	if _, err := s.GetTaskItems(stranger, 2); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("get foreign items: err = %v, want ErrNotFound", err)
	}
	if _, err := s.CreateTaskItem(stranger, 2, "x"); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("add to foreign task: err = %v, want ErrNotFound", err)
	}
	if _, err := s.PatchTaskItem(stranger, 2, 1, domain.TaskItemPatch{Done: domain.Nullable[bool]{Value: &done, Set: true}}); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("patch foreign item: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteTaskItem(stranger, 2, 1); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("delete foreign item: err = %v, want ErrNotFound", err)
	}
	if repo.items[1].Done || len(repo.items) != 1 {
		t.Fatalf("foreign item changed: %+v", repo.items)
	}

	// Участник общего списка 200 работает с чеклистом наравне с владельцем.
	member := asActor(30, false)
	if _, err := s.PatchTaskItem(member, 2, 1, domain.TaskItemPatch{Done: domain.Nullable[bool]{Value: &done, Set: true}}); err != nil {
		t.Fatalf("member patch item: %v", err)
	}
	if !repo.items[1].Done {
		t.Fatal("item not checked")
	}
}

func TestItemOfAnotherTaskIsHidden(t *testing.T) {
	repo := newRepo()
	repo.items = map[int]domain.TaskItem{1: {ID: 1, Version: 1, TaskID: 2, Title: "из задачи 2"}}

	// Задача 1 своя, но пункт 1 принадлежит задаче 2.
	if err := newService(repo).DeleteTaskItem(asActor(10, false), 1, 1); !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCreateTaskItem(t *testing.T) {
	repo := newRepo()
	s := newService(repo)

	if _, err := s.CreateTaskItem(asActor(10, false), 1, ""); !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("empty title: err = %v, want ErrInvalidArgument", err)
	}

	item, err := s.CreateTaskItem(asActor(10, false), 1, "Молоко")
	if err != nil {
		t.Fatal(err)
	}
	if item.TaskID != 1 || item.Title != "Молоко" || item.Done {
		t.Fatalf("item = %+v", item)
	}
}

func TestPatchTaskItemVersionConflict(t *testing.T) {
	repo := newRepo()
	repo.items = map[int]domain.TaskItem{1: {ID: 1, Version: 3, TaskID: 1, Title: "Хлеб"}}
	done, stale := true, 2

	_, err := newService(repo).PatchTaskItem(asActor(10, false), 1, 1, domain.TaskItemPatch{
		Done:            domain.Nullable[bool]{Value: &done, Set: true},
		ExpectedVersion: &stale,
	})
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestNextOccurrenceGetsChecklist(t *testing.T) {
	repo := newRepo()
	due := time.Now().Add(-time.Hour)
	repo.tasks[1] = domain.Task{ID: 1, Version: 1, Title: "Уборка", CreatedAt: due.Add(-time.Hour), AuthorUserID: 10, ListID: 100,
		DueAt: &due, Repeat: &domain.Recurrence{Kind: domain.RepeatWeekly, Weekdays: []int{6}}, ItemsTotal: 3, ItemsDone: 3}
	done := true

	if _, err := newService(repo).PatchTask(asActor(10, false), 1, domain.TaskPatch{Completed: domain.Nullable[bool]{Value: &done, Set: true}}); err != nil {
		t.Fatal(err)
	}
	if len(repo.copied) != 1 || repo.copied[0][0] != 1 {
		t.Fatalf("checklist not copied to next occurrence: %v", repo.copied)
	}
}

func TestNextOccurrenceWithoutChecklist(t *testing.T) {
	repo := newRepo()
	due := time.Now().Add(-time.Hour)
	repo.tasks[1] = domain.Task{ID: 1, Version: 1, Title: "Планёрка", CreatedAt: due.Add(-time.Hour), AuthorUserID: 10, ListID: 100,
		DueAt: &due, Repeat: &domain.Recurrence{Kind: domain.RepeatDaily}}
	done := true

	if _, err := newService(repo).PatchTask(asActor(10, false), 1, domain.TaskPatch{Completed: domain.Nullable[bool]{Value: &done, Set: true}}); err != nil {
		t.Fatal(err)
	}
	if len(repo.copied) != 0 {
		t.Fatalf("copied = %v, want nothing", repo.copied)
	}
}
