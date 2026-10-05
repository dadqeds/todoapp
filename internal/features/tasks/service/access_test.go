package tasks_service

import (
	"context"
	"errors"
	"testing"
	"time"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type fakeTasksRepository struct {
	tasks     map[int]domain.Task
	created   domain.Task
	deleted   []int
	gotUserID *int
}

func (f *fakeTasksRepository) CreateTask(_ context.Context, task domain.Task) (domain.Task, error) {
	f.created = task
	return task, nil
}

func (f *fakeTasksRepository) GetTasks(_ context.Context, userID *int, _ int, _ int) ([]domain.Task, error) {
	f.gotUserID = userID
	return nil, nil
}

func (f *fakeTasksRepository) GetTask(_ context.Context, id int) (domain.Task, error) {
	task, ok := f.tasks[id]
	if !ok {
		return domain.Task{}, core_errors.ErrNotFound
	}
	return task, nil
}

func (f *fakeTasksRepository) DeleteTask(_ context.Context, id int) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeTasksRepository) PatchTask(_ context.Context, _ int, task domain.Task) (domain.Task, error) {
	return task, nil
}

func asActor(id int, isAdmin bool) context.Context {
	return core_auth.ToContext(context.Background(), core_auth.Actor{User: domain.User{ID: id}, IsAdmin: isAdmin})
}

func newRepo() *fakeTasksRepository {
	now := time.Now()
	return &fakeTasksRepository{tasks: map[int]domain.Task{
		1: {ID: 1, Version: 1, Title: "mine", CreatedAt: now, AuthorUserID: 10},
		2: {ID: 2, Version: 1, Title: "foreign", CreatedAt: now, AuthorUserID: 20},
	}}
}

func TestUserCannotTouchForeignTasks(t *testing.T) {
	repo := newRepo()
	s := NewTasksService(repo)
	ctx := asActor(10, false)
	title := "hacked"
	patch := domain.TaskPatch{Title: domain.Nullable[string]{Value: &title, Set: true}}

	if _, err := s.GetTask(ctx, 2); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("get foreign: err = %v, want ErrNotFound", err)
	}
	if _, err := s.PatchTask(ctx, 2, patch); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("patch foreign: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteTask(ctx, 2); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("delete foreign: err = %v, want ErrNotFound", err)
	}
	if len(repo.deleted) != 0 {
		t.Errorf("foreign task was deleted: %v", repo.deleted)
	}

	if _, err := s.GetTask(ctx, 1); err != nil {
		t.Errorf("get own: %v", err)
	}
	if err := s.DeleteTask(ctx, 1); err != nil {
		t.Errorf("delete own: %v", err)
	}
}

func TestAdminCanTouchAnyTask(t *testing.T) {
	s := NewTasksService(newRepo())

	if _, err := s.GetTask(asActor(10, true), 2); err != nil {
		t.Fatalf("admin get foreign: %v", err)
	}
}

func TestGetTasksScope(t *testing.T) {
	other := 20

	repo := newRepo()
	if _, err := NewTasksService(repo).GetTasks(asActor(10, false), &other, nil, nil); err != nil {
		t.Fatal(err)
	}
	if repo.gotUserID == nil || *repo.gotUserID != 10 {
		t.Fatalf("user sees user_id = %v, want 10", repo.gotUserID)
	}

	repo = newRepo()
	if _, err := NewTasksService(repo).GetTasks(asActor(10, true), nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if repo.gotUserID != nil {
		t.Fatalf("admin without filter got user_id = %v, want nil", *repo.gotUserID)
	}
}

func TestCreateTaskAuthor(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		requestAuthor int
		wantAuthor    int
	}{
		{"user cannot create for another user", asActor(10, false), 20, 10},
		{"user without author", asActor(10, false), 0, 10},
		{"admin chooses author", asActor(10, true), 20, 20},
		{"admin without author", asActor(10, true), 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()
			task := domain.NewTaskUninitialized("t", nil, tt.requestAuthor)

			if _, err := NewTasksService(repo).CreateTask(tt.ctx, task); err != nil {
				t.Fatal(err)
			}
			if repo.created.AuthorUserID != tt.wantAuthor {
				t.Fatalf("author = %d, want %d", repo.created.AuthorUserID, tt.wantAuthor)
			}
		})
	}
}
