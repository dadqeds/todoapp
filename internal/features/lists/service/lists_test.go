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
	members      map[int]map[int]bool
	deleted      []int
	ensuredFor   []int
	gotOwner     *int
	createdOwner int
}

// Пользователь 10 владеет списками 1 и 2, 20 — списком 3, в котором участвует 30.
func newRepo() *fakeRepo {
	code := "invite-code-3"
	return &fakeRepo{
		lists: map[int]domain.List{
			1: {ID: 1, OwnerUserID: 10, IsDefault: true, Title: "Личное", Color: "coral"},
			2: {ID: 2, OwnerUserID: 10, Title: "Дом", Color: "green"},
			3: {ID: 3, OwnerUserID: 20, Title: "Чужой", Color: "blue", InviteCode: &code},
		},
		members: map[int]map[int]bool{3: {30: true}},
	}
}

func (f *fakeRepo) GetList(_ context.Context, id int) (domain.List, error) {
	l, ok := f.lists[id]
	if !ok {
		return domain.List{}, core_errors.ErrNotFound
	}
	return l, nil
}

func (f *fakeRepo) GetListsForUser(_ context.Context, owner int) ([]domain.ListSummary, error) {
	f.gotOwner = &owner
	return nil, nil
}

func (f *fakeRepo) IsListMember(_ context.Context, listID, userID int) (bool, error) {
	return f.members[listID][userID], nil
}

func (f *fakeRepo) AddMember(_ context.Context, listID, userID int) error {
	if f.members[listID] == nil {
		f.members[listID] = map[int]bool{}
	}
	f.members[listID][userID] = true
	return nil
}

func (f *fakeRepo) RemoveMember(_ context.Context, listID, userID int) error {
	if !f.members[listID][userID] {
		return core_errors.ErrNotFound
	}
	delete(f.members[listID], userID)
	return nil
}

func (f *fakeRepo) SetInviteCode(_ context.Context, listID int, code *string) (domain.List, error) {
	l := f.lists[listID]
	l.InviteCode = code
	f.lists[listID] = l
	return l, nil
}

func (f *fakeRepo) GetListByInviteCode(_ context.Context, code string) (domain.List, error) {
	for _, l := range f.lists {
		if l.InviteCode != nil && *l.InviteCode == code {
			return l, nil
		}
	}
	return domain.List{}, core_errors.ErrNotFound
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

func TestMemberCanViewButNotManage(t *testing.T) {
	s := NewListsService(newRepo())
	color := "pink"
	patch := domain.ListPatch{Color: domain.Nullable[string]{Value: &color, Set: true}}

	if _, err := s.PatchList(asActor(30, false), 3, patch); !errors.Is(err, core_errors.ErrForbidden) {
		t.Errorf("member patch: err = %v, want ErrForbidden", err)
	}
	if err := s.DeleteList(asActor(30, false), 3); !errors.Is(err, core_errors.ErrForbidden) {
		t.Errorf("member delete: err = %v, want ErrForbidden", err)
	}
	if _, err := s.CreateInvite(asActor(30, false), 3); !errors.Is(err, core_errors.ErrForbidden) {
		t.Errorf("member invite: err = %v, want ErrForbidden", err)
	}
	if _, err := s.PatchList(asActor(40, false), 3, patch); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("stranger patch: err = %v, want ErrNotFound", err)
	}
}

func TestInvites(t *testing.T) {
	repo := newRepo()
	s := NewListsService(repo)
	owner := asActor(10, false)

	if _, err := s.CreateInvite(owner, 1); !errors.Is(err, core_errors.ErrConflict) {
		t.Errorf("default list invite: err = %v, want ErrConflict", err)
	}

	list, err := s.CreateInvite(owner, 2)
	if err != nil {
		t.Fatal(err)
	}
	if list.InviteCode == nil || len(*list.InviteCode) != 16 {
		t.Fatalf("invite code = %v, want 16 chars", list.InviteCode)
	}
	first := *list.InviteCode

	again, _ := s.CreateInvite(owner, 2)
	if *again.InviteCode == first {
		t.Fatal("new invite must replace the old code")
	}

	if _, err := s.JoinList(asActor(50, false), first); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("join by old code: err = %v, want ErrNotFound", err)
	}

	joined, err := s.JoinList(asActor(50, false), *again.InviteCode)
	if err != nil || joined.ID != 2 || !repo.members[2][50] {
		t.Fatalf("join: list %d, err %v, members %v", joined.ID, err, repo.members[2])
	}

	if _, err := s.JoinList(owner, *again.InviteCode); err != nil || repo.members[2][10] {
		t.Fatalf("owner join must be a no-op: err %v, members %v", err, repo.members[2])
	}

	if err := s.RevokeInvite(owner, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.JoinList(asActor(60, false), *again.InviteCode); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("join after revoke: err = %v, want ErrNotFound", err)
	}
}

func TestRemoveMember(t *testing.T) {
	repo := newRepo()
	repo.members[3][31] = true
	s := NewListsService(repo)

	if err := s.RemoveMember(asActor(30, false), 3, 31); !errors.Is(err, core_errors.ErrForbidden) {
		t.Errorf("member removes other: err = %v, want ErrForbidden", err)
	}
	if err := s.RemoveMember(asActor(20, false), 3, 20); !errors.Is(err, core_errors.ErrConflict) {
		t.Errorf("owner leaves: err = %v, want ErrConflict", err)
	}
	if err := s.RemoveMember(asActor(30, false), 3, 30); err != nil || repo.members[3][30] {
		t.Errorf("member leaves: err %v, members %v", err, repo.members[3])
	}
	if err := s.RemoveMember(asActor(20, false), 3, 31); err != nil || repo.members[3][31] {
		t.Errorf("owner removes member: err %v, members %v", err, repo.members[3])
	}
}
