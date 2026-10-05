package core_http_middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
)

type fakeResolver struct{}

func (fakeResolver) ResolveTelegramUser(_ context.Context, tg core_auth.TelegramUser) (domain.User, error) {
	return domain.User{ID: int(tg.ID) * 10}, nil
}

func TestTelegramAuthRejectsMissingOrInvalidInitData(t *testing.T) {
	h := TelegramAuth(core_auth.Config{TelegramBotToken: "1:x"}, fakeResolver{})(okHandler)

	for name, header := range map[string]string{
		"no header":     "",
		"wrong scheme":  "Bearer abc",
		"bad signature": "tma auth_date=1&hash=00&user=%7B%22id%22%3A1%7D",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("code = %d, want 401", rec.Code)
			}
		})
	}
}

func TestLocalAuthActsAsAdmin(t *testing.T) {
	var actor core_auth.Actor
	h := LocalAuth(5, fakeResolver{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, _ = core_auth.FromContext(r.Context())
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if actor.User.ID != 50 || !actor.IsAdmin {
		t.Fatalf("actor = %+v", actor)
	}
}
