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
	gotFilter domain.TaskFilter
}

func (f *fakeTasksRepository) CreateTask(_ context.Context, task domain.Task) (domain.Task, error) {
	f.created = task
	return task, nil
}

func (f *fakeTasksRepository) GetTasks(_ context.Context, filter domain.TaskFilter, _ int, _ int) ([]domain.Task, error) {
	f.gotFilter = filter
	return nil, nil
}

// Списки: 100 — по умолчанию у пользователя 10, 101 — ещё один у 10, 200 — у 20.
// В списке 200 участвует пользователь 30.
type fakeListsRepository struct{}

func (fakeListsRepository) IsListMember(_ context.Context, listID, userID int) (bool, error) {
	return listID == 200 && userID == 30, nil
}

var fakeLists = map[int]domain.List{
	100: {ID: 100, OwnerUserID: 10, IsDefault: true},
	101: {ID: 101, OwnerUserID: 10},
	200: {ID: 200, OwnerUserID: 20, IsDefault: true},
}

func (fakeListsRepository) GetList(_ context.Context, id int) (domain.List, error) {
	l, ok := fakeLists[id]
	if !ok {
		return domain.List{}, core_errors.ErrNotFound
	}
	return l, nil
}

func (fakeListsRepository) GetOrCreateDefaultList(_ context.Context, owner int) (domain.List, error) {
	for _, l := range fakeLists {
		if l.OwnerUserID == owner && l.IsDefault {
			return l, nil
		}
	}
	return domain.List{}, core_errors.ErrNotFound
}

func newService(repo *fakeTasksRepository) *TasksService {
	return NewTasksService(repo, fakeListsRepository{})
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
		1: {ID: 1, Version: 1, Title: "mine", CreatedAt: now, AuthorUserID: 10, ListID: 100},
		2: {ID: 2, Version: 1, Title: "foreign", CreatedAt: now, AuthorUserID: 20, ListID: 200},
		// Задача 10-го в списке 200, из которого он не состоит (например, вышел).
		3: {ID: 3, Version: 1, Title: "left", CreatedAt: now, AuthorUserID: 10, ListID: 200},
	}}
}

func TestUserCannotTouchForeignTasks(t *testing.T) {
	repo := newRepo()
	s := newService(repo)
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
	s := newService(newRepo())

	if _, err := s.GetTask(asActor(10, true), 2); err != nil {
		t.Fatalf("admin get foreign: %v", err)
	}
}

func TestGetTasksScope(t *testing.T) {
	other := 20

	repo := newRepo()
	if _, err := newService(repo).GetTasks(asActor(10, false), domain.TaskFilter{AuthorUserID: &other}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if repo.gotFilter.AuthorUserID != nil || repo.gotFilter.AccessibleToUserID == nil || *repo.gotFilter.AccessibleToUserID != 10 {
		t.Fatalf("user filter = %+v, want only lists accessible to 10", repo.gotFilter)
	}

	repo = newRepo()
	if _, err := newService(repo).GetTasks(asActor(10, true), domain.TaskFilter{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if repo.gotFilter.AuthorUserID != nil || repo.gotFilter.AccessibleToUserID == nil {
		t.Fatalf("admin without user_id: filter = %+v, want own and shared lists", repo.gotFilter)
	}

	repo = newRepo()
	if _, err := newService(repo).GetTasks(asActor(10, true), domain.TaskFilter{AuthorUserID: &other}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if repo.gotFilter.AuthorUserID == nil || *repo.gotFilter.AuthorUserID != other || repo.gotFilter.AccessibleToUserID != nil {
		t.Fatalf("admin with user_id: filter = %+v, want author %d", repo.gotFilter, other)
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
			task := domain.NewTaskUninitialized("t", nil, tt.requestAuthor, 0, nil, false)

			if _, err := newService(repo).CreateTask(tt.ctx, task); err != nil {
				t.Fatal(err)
			}
			if repo.created.AuthorUserID != tt.wantAuthor {
				t.Fatalf("author = %d, want %d", repo.created.AuthorUserID, tt.wantAuthor)
			}
		})
	}
}

func TestCreateTaskList(t *testing.T) {
	tests := []struct {
		name     string
		listID   int
		wantList int
		wantErr  error
	}{
		{"default list when not set", 0, 100, nil},
		{"own list", 101, 101, nil},
		{"foreign list is hidden", 200, 0, core_errors.ErrNotFound},
		{"missing list", 999, 0, core_errors.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepo()
			_, err := newService(repo).CreateTask(asActor(10, false), domain.NewTaskUninitialized("t", nil, 0, tt.listID, nil, false))

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if repo.created.ListID != tt.wantList {
				t.Fatalf("list = %d, want %d", repo.created.ListID, tt.wantList)
			}
		})
	}
}

func TestPatchTaskCannotMoveToForeignList(t *testing.T) {
	foreign := 200
	patch := domain.TaskPatch{ListID: domain.Nullable[int]{Value: &foreign, Set: true}}

	_, err := newService(newRepo()).PatchTask(asActor(10, false), 1, patch)
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSharedListMember(t *testing.T) {
	repo := newRepo()
	s := newService(repo)
	member := asActor(30, false)

	if _, err := s.GetTask(member, 2); err != nil {
		t.Errorf("member reads task in shared list: %v", err)
	}
	if err := s.DeleteTask(member, 2); err != nil {
		t.Errorf("member deletes task in shared list: %v", err)
	}
	if _, err := s.GetTask(member, 1); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("member reads task from not shared list: err = %v, want ErrNotFound", err)
	}

	if _, err := s.CreateTask(member, domain.NewTaskUninitialized("t", nil, 0, 200, nil, false)); err != nil {
		t.Fatalf("member creates task in shared list: %v", err)
	}
	if repo.created.AuthorUserID != 30 || repo.created.ListID != 200 {
		t.Fatalf("created = %+v, want author 30 in list 200", repo.created)
	}
}

func TestAuthorWhoLeftListLosesAccess(t *testing.T) {
	if _, err := newService(newRepo()).GetTask(asActor(10, false), 3); !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
