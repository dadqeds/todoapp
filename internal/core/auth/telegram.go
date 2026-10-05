// Package core_auth проверяет данные запуска Telegram Mini App и хранит
// текущего пользователя запроса в контексте.
package core_auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// TelegramUser — поля пользователя из initData, которые нам нужны.
type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// FullName собирает отображаемое имя: «Имя Фамилия», иначе username, иначе id.
func (u TelegramUser) FullName() string {
	name := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	if name == "" {
		name = strings.TrimSpace(u.Username)
	}
	if name == "" {
		name = fmt.Sprintf("Пользователь %d", u.ID)
	}

	runes := []rune(name)
	if len(runes) > 100 {
		name = string(runes[:100])
	}

	return name
}

// ValidateInitData проверяет подпись initData токеном бота и её свежесть
// по алгоритму из документации Telegram:
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func ValidateInitData(
	initData string,
	botToken string,
	maxAge time.Duration,
	now time.Time,
) (TelegramUser, error) {
	if botToken == "" {
		return TelegramUser{}, fmt.Errorf("telegram bot token is not configured: %w", core_errors.ErrUnauthenticated)
	}

	values, err := url.ParseQuery(initData)
	if err != nil {
		return TelegramUser{}, fmt.Errorf("parse init data: %v: %w", err, core_errors.ErrUnauthenticated)
	}

	hash := values.Get("hash")
	if hash == "" {
		return TelegramUser{}, fmt.Errorf("init data has no hash: %w", core_errors.ErrUnauthenticated)
	}

	pairs := make([]string, 0, len(values))
	for key := range values {
		if key == "hash" {
			continue
		}
		pairs = append(pairs, key+"="+values.Get(key))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	secretKey := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	expected := hmacSHA256(secretKey, []byte(dataCheckString))

	got, err := hex.DecodeString(hash)
	if err != nil || !hmac.Equal(got, expected) {
		return TelegramUser{}, fmt.Errorf("invalid init data signature: %w", core_errors.ErrUnauthenticated)
	}

	authDateUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return TelegramUser{}, fmt.Errorf("invalid auth_date: %w", core_errors.ErrUnauthenticated)
	}
	if maxAge > 0 && now.Sub(time.Unix(authDateUnix, 0)) > maxAge {
		return TelegramUser{}, fmt.Errorf("init data expired: %w", core_errors.ErrUnauthenticated)
	}

	var user TelegramUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return TelegramUser{}, fmt.Errorf("init data has no valid user: %w", core_errors.ErrUnauthenticated)
	}

	return user, nil
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
