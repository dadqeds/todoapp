package lists_service

import (
	"context"
	"errors"
	"testing"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type fakeRepo struct {
	lists        map[int]domain.List
	deleted      []int
	ensuredFor   []int
	gotOwner     *int
	createdOwner int
}

func newRepo() *fakeRepo {
	return &fakeRepo{lists: map[int]domain.List{
		1: {ID: 1, OwnerUserID: 10, IsDefault: true, Title: "Личное", Color: "coral"},
		2: {ID: 2, OwnerUserID: 10, Title: "Дом", Color: "green"},
		3: {ID: 3, OwnerUserID: 20, Title: "Чужой", Color: "blue"},
	}}
}

func (f *fakeRepo) GetList(_ context.Context, id int) (domain.List, error) {
	l, ok := f.lists[id]
	if !ok {
		return domain.List{}, core_errors.ErrNotFound
	}
	return l, nil
}

func (f *fakeRepo) GetLists(_ context.Context, owner *int) ([]domain.ListSummary, error) {
	f.gotOwner = owner
	return nil, nil
}

func (f *fakeRepo) GetOrCreateDefaultList(_ context.Context, owner int) (domain.List, error) {
	f.ensuredFor = append(f.ensuredFor, owner)
	return domain.List{}, nil
}

func (f *fakeRepo) CreateList(_ context.Context, l domain.List) (domain.List, error) {
	f.createdOwner = l.OwnerUserID
	return l, nil
}

func (f *fakeRepo) PatchList(_ context.Context, _ int, l domain.List) (domain.List, error) {
	return l, nil
}

func (f *fakeRepo) DeleteList(_ context.Context, id int) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func asActor(id int, isAdmin bool) context.Context {
	return core_auth.ToContext(context.Background(), core_auth.Actor{User: domain.User{ID: id}, IsAdmin: isAdmin})
}

func TestGetListsScope(t *testing.T) {
	other := 20

	repo := newRepo()
	if _, err := NewListsService(repo).GetLists(asActor(10, false), &other); err != nil {
		t.Fatal(err)
	}
	if *repo.gotOwner != 10 || repo.ensuredFor[0] != 10 {
		t.Fatalf("user got lists of %d (ensured %v), want own", *repo.gotOwner, repo.ensuredFor)
	}

	repo = newRepo()
	if _, err := NewListsService(repo).GetLists(asActor(10, true), &other); err != nil {
		t.Fatal(err)
	}
	if *repo.gotOwner != 20 {
		t.Fatalf("admin got lists of %d, want 20", *repo.gotOwner)
	}
}

func TestCreateListOwnerIsActor(t *testing.T) {
	repo := newRepo()
	list := domain.NewListUninitialized("Работа", "violet", 20)

	if _, err := NewListsService(repo).CreateList(asActor(10, false), list); err != nil {
		t.Fatal(err)
	}
	if repo.createdOwner != 10 {
		t.Fatalf("owner = %d, want 10", repo.createdOwner)
	}
}

func TestDeleteList(t *testing.T) {
	repo := newRepo()
	s := NewListsService(repo)
	ctx := asActor(10, false)

	if err := s.DeleteList(ctx, 1); !errors.Is(err, core_errors.ErrConflict) {
		t.Errorf("default list: err = %v, want ErrConflict", err)
	}
	if err := s.DeleteList(ctx, 3); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("foreign list: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteList(ctx, 2); err != nil {
		t.Errorf("own list: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != 2 {
		t.Fatalf("deleted = %v, want [2]", repo.deleted)
	}
}

func TestPatchForeignListIsHidden(t *testing.T) {
	color := "pink"
	_, err := NewListsService(newRepo()).PatchList(asActor(10, false), 3, domain.ListPatch{Color: domain.Nullable[string]{Value: &color, Set: true}})
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
