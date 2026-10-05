package users_service

import (
	"context"
	"errors"
	"testing"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type fakeUsersRepository struct {
	UsersRepository
	users   map[int]domain.User
	created []domain.User
}

func (f *fakeUsersRepository) GetUser(_ context.Context, id int) (domain.User, error) {
	user, ok := f.users[id]
	if !ok {
		return domain.User{}, core_errors.ErrNotFound
	}
	return user, nil
}

func (f *fakeUsersRepository) GetUsers(context.Context, int, int) ([]domain.User, error) {
	return nil, nil
}

func (f *fakeUsersRepository) DeleteUser(context.Context, int) error { return nil }

func (f *fakeUsersRepository) GetUserByTelegramID(_ context.Context, telegramID int64) (domain.User, error) {
	for _, u := range f.users {
		if u.TelegramID != nil && *u.TelegramID == telegramID {
			return u, nil
		}
	}
	return domain.User{}, core_errors.ErrNotFound
}

func (f *fakeUsersRepository) PatchUser(_ context.Context, id int, user domain.User) (domain.User, error) {
	f.users[id] = user
	return user, nil
}

func (f *fakeUsersRepository) CreateTelegramUser(_ context.Context, user domain.User) (domain.User, error) {
	user.ID = 100 + len(f.created)
	f.created = append(f.created, user)
	return user, nil
}

func asActor(id int, isAdmin bool) context.Context {
	return core_auth.ToContext(context.Background(), core_auth.Actor{User: domain.User{ID: id}, IsAdmin: isAdmin})
}

func TestUsersAccess(t *testing.T) {
	repo := &fakeUsersRepository{users: map[int]domain.User{
		1: {ID: 1, FullName: "me"},
		2: {ID: 2, FullName: "other"},
	}}
	s := NewUserService(repo)
	user := asActor(1, false)

	if _, err := s.GetUser(user, 1); err != nil {
		t.Errorf("get self: %v", err)
	}
	if _, err := s.GetUser(user, 2); !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("get other: err = %v, want ErrNotFound", err)
	}
	if _, err := s.GetUsers(user, nil, nil); !errors.Is(err, core_errors.ErrForbidden) {
		t.Errorf("list: err = %v, want ErrForbidden", err)
	}
	if _, err := s.CreateUser(user, domain.User{FullName: "x"}); !errors.Is(err, core_errors.ErrForbidden) {
		t.Errorf("create: err = %v, want ErrForbidden", err)
	}
	if err := s.DeleteUser(user, 2); !errors.Is(err, core_errors.ErrForbidden) {
		t.Errorf("delete: err = %v, want ErrForbidden", err)
	}

	admin := asActor(1, true)
	if _, err := s.GetUser(admin, 2); err != nil {
		t.Errorf("admin get other: %v", err)
	}
	if _, err := s.GetUsers(admin, nil, nil); err != nil {
		t.Errorf("admin list: %v", err)
	}
}

func TestResolveTelegramUser(t *testing.T) {
	tgID := int64(555)
	repo := &fakeUsersRepository{users: map[int]domain.User{
		7: {ID: 7, FullName: "Уже есть", TelegramID: &tgID},
	}}
	s := NewUserService(repo)

	existing, err := s.ResolveTelegramUser(context.Background(), core_auth.TelegramUser{ID: 555, FirstName: "Новое имя"})
	if err != nil {
		t.Fatal(err)
	}
	if existing.ID != 7 || existing.FullName != "Уже есть" || len(repo.created) != 0 {
		t.Fatalf("existing user must be reused unchanged: %+v, created %d", existing, len(repo.created))
	}

	created, err := s.ResolveTelegramUser(context.Background(), core_auth.TelegramUser{ID: 777, FirstName: "Ян"})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 || created.FullName != "Ян" || created.TelegramID == nil || *created.TelegramID != 777 {
		t.Fatalf("new user not created correctly: %+v", created)
	}
}

func TestResolveReplacesLocalPlaceholderName(t *testing.T) {
	tgID := int64(42)
	repo := &fakeUsersRepository{users: map[int]domain.User{
		11: {ID: 11, FullName: core_auth.LocalUserFullName, TelegramID: &tgID},
	}}
	s := NewUserService(repo)

	local, _ := s.ResolveTelegramUser(context.Background(), core_auth.TelegramUser{ID: 42, FirstName: core_auth.LocalUserFullName})
	if local.FullName != core_auth.LocalUserFullName {
		t.Fatalf("local login must keep placeholder, got %q", local.FullName)
	}

	user, err := s.ResolveTelegramUser(context.Background(), core_auth.TelegramUser{ID: 42, FirstName: "Оганес"})
	if err != nil {
		t.Fatal(err)
	}
	if user.FullName != "Оганес" {
		t.Fatalf("name = %q, want name from Telegram", user.FullName)
	}

	again, _ := s.ResolveTelegramUser(context.Background(), core_auth.TelegramUser{ID: 42, FirstName: "Другое"})
	if again.FullName != "Оганес" {
		t.Fatalf("real name must not be overwritten, got %q", again.FullName)
	}
}
