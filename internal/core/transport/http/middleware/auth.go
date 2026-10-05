package core_http_middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

const telegramAuthScheme = "tma "

// UserResolver находит пользователя по аккаунту Telegram, создавая его при первом входе.
type UserResolver interface {
	ResolveTelegramUser(ctx context.Context, tgUser core_auth.TelegramUser) (domain.User, error)
}

// TelegramAuth пускает только запросы с валидным initData Telegram Mini App
// в заголовке `Authorization: tma <initData>`.
func TelegramAuth(config core_auth.Config, resolver UserResolver) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, telegramAuthScheme) {
				authError(w, r, fmt.Errorf("missing telegram init data: %w", core_errors.ErrUnauthenticated))
				return
			}

			tgUser, err := core_auth.ValidateInitData(
				strings.TrimPrefix(header, telegramAuthScheme),
				config.TelegramBotToken,
				config.InitDataMaxAge,
				time.Now(),
			)
			if err != nil {
				authError(w, r, err)
				return
			}

			serveAs(w, r, next, resolver, tgUser, config.IsAdmin(tgUser.ID))
		})
	}
}

// LocalAuth выполняет все запросы от имени заданного аккаунта Telegram с правами
// администратора. Используется только на локальном адресе (127.0.0.1).
func LocalAuth(telegramID int64, resolver UserResolver) Middleware {
	tgUser := core_auth.TelegramUser{ID: telegramID, FirstName: core_auth.LocalUserFullName}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serveAs(w, r, next, resolver, tgUser, true)
		})
	}
}

func serveAs(
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	resolver UserResolver,
	tgUser core_auth.TelegramUser,
	isAdmin bool,
) {
	user, err := resolver.ResolveTelegramUser(r.Context(), tgUser)
	if err != nil {
		core_http_response.NewHTTPResponseHandler(core_logger.FromContext(r.Context()), w).
			ErrorResponse(err, "failed to resolve telegram user")
		return
	}

	ctx := core_auth.ToContext(r.Context(), core_auth.Actor{User: user, IsAdmin: isAdmin})
	next.ServeHTTP(w, r.WithContext(ctx))
}

func authError(w http.ResponseWriter, r *http.Request, err error) {
	core_http_response.NewHTTPResponseHandler(core_logger.FromContext(r.Context()), w).
		ErrorResponse(err, "authentication required")
}
