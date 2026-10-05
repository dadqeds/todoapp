package core_auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

const testBotToken = "123456:TEST-token"

// signInitData собирает initData так же, как это делает Telegram.
func signInitData(t *testing.T, token string, fields map[string]string) string {
	t.Helper()

	pairs := make([]string, 0, len(fields))
	values := url.Values{}
	for k, v := range fields {
		pairs = append(pairs, k+"="+v)
		values.Set(k, v)
	}
	sort.Strings(pairs)

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(pairs, "\n")))

	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}

func TestValidateInitData(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	fields := func(authDate time.Time) map[string]string {
		return map[string]string{
			"auth_date": strconv.FormatInt(authDate.Unix(), 10),
			"query_id":  "AAH",
			"user":      `{"id":42,"first_name":"Иван","last_name":"Петров","username":"ivan"}`,
		}
	}

	t.Run("valid", func(t *testing.T) {
		user, err := ValidateInitData(signInitData(t, testBotToken, fields(now.Add(-time.Minute))), testBotToken, 24*time.Hour, now)
		if err != nil {
			t.Fatal(err)
		}
		if user.ID != 42 || user.FullName() != "Иван Петров" {
			t.Fatalf("user = %+v", user)
		}
	})

	invalid := map[string]string{
		"wrong token":   signInitData(t, "other:token", fields(now)),
		"expired":       signInitData(t, testBotToken, fields(now.Add(-25*time.Hour))),
		"no hash":       "auth_date=1&user=%7B%22id%22%3A42%7D",
		"garbage":       "%%%",
		"empty":         "",
		"no user":       signInitData(t, testBotToken, map[string]string{"auth_date": strconv.FormatInt(now.Unix(), 10)}),
		"tampered user": strings.Replace(signInitData(t, testBotToken, fields(now)), "42", "43", 1),
	}
	for name, initData := range invalid {
		t.Run(name, func(t *testing.T) {
			_, err := ValidateInitData(initData, testBotToken, 24*time.Hour, now)
			if !errors.Is(err, core_errors.ErrUnauthenticated) {
				t.Fatalf("err = %v, want ErrUnauthenticated", err)
			}
		})
	}

	t.Run("bot token not configured", func(t *testing.T) {
		_, err := ValidateInitData(signInitData(t, testBotToken, fields(now)), "", 24*time.Hour, now)
		if !errors.Is(err, core_errors.ErrUnauthenticated) {
			t.Fatalf("err = %v, want ErrUnauthenticated", err)
		}
	})
}

func TestTelegramUserFullName(t *testing.T) {
	tests := []struct {
		user TelegramUser
		want string
	}{
		{TelegramUser{ID: 1, FirstName: "Ян"}, "Ян"},
		{TelegramUser{ID: 1, FirstName: " ", Username: "nick"}, "nick"},
		{TelegramUser{ID: 7}, "Пользователь 7"},
		{TelegramUser{ID: 1, FirstName: strings.Repeat("я", 120)}, strings.Repeat("я", 100)},
	}

	for _, tt := range tests {
		if got := tt.user.FullName(); got != tt.want {
			t.Errorf("FullName(%+v) = %q, want %q", tt.user, got, tt.want)
		}
	}
}
