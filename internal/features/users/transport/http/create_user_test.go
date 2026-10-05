package users_transport_http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

type fakeUsersService struct {
	UsersService
	createErr error
}

func (f fakeUsersService) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	if f.createErr != nil {
		return domain.User{}, f.createErr
	}
	u.ID, u.Version = 1, 1
	return u, nil
}

func TestCreateUserErrorWritesSingleResponse(t *testing.T) {
	h := NewUsersHTTPHandler(fakeUsersService{
		createErr: fmt.Errorf("phone: %w", core_errors.ErrInvalidArgument),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"full_name":"Ivan"}`))

	h.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}

	// Раньше после ошибки в тело дописывался второй JSON с пустым пользователем.
	dec := json.NewDecoder(rec.Body)
	var body core_http_response.ErrorResponse
	if err := dec.Decode(&body); err != nil {
		t.Fatal(err)
	}
	if dec.More() {
		t.Fatalf("response contains more than one JSON document")
	}
}

func TestCreateUserRejectsHugeBody(t *testing.T) {
	h := NewUsersHTTPHandler(fakeUsersService{})

	huge := `{"full_name":"` + strings.Repeat("a", 2<<20) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(huge))

	h.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}
