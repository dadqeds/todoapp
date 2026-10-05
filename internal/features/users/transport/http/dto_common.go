package users_transport_http

import (
	"fmt"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

type UserDTOResponse struct {
	ID          int     `json:"id"            example:"10"`
	Version     int     `json:"version"       example:"3"`
	FullName    string  `json:"full_name"     example:"Ivan Ivanov"`
	PhoneNumber *string `json:"phone_number"  example:"+79998887766"`
	TelegramID  *int64  `json:"telegram_id"   example:"123456789"`
	Timezone    string  `json:"timezone"      example:"Europe/Moscow"`

	RemindEnabled bool   `json:"remind_enabled" example:"true"`
	DigestEnabled bool   `json:"digest_enabled" example:"true"`
	DigestTime    string `json:"digest_time"    example:"08:30"`
}

func formatDigestTime(minute int) string {
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60)
}

// parseDigestTime разбирает «ЧЧ:ММ» в минуты от полуночи.
func parseDigestTime(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("digest_time %q must be HH:MM: %w", s, core_errors.ErrInvalidArgument)
	}
	return t.Hour()*60 + t.Minute(), nil
}

func userDTOFromDomain(user domain.User) UserDTOResponse {
	return UserDTOResponse{
		ID:          user.ID,
		Version:     user.Version,
		FullName:    user.FullName,
		PhoneNumber: user.PhoneNumber,
		TelegramID:  user.TelegramID,
		Timezone:    user.Timezone,

		RemindEnabled: user.RemindEnabled,
		DigestEnabled: user.DigestEnabled,
		DigestTime:    formatDigestTime(user.DigestMinute),
	}
}

func usersDTOFromDomains(users []domain.User) []UserDTOResponse {
	usersDTO := make([]UserDTOResponse, len(users))

	for i, user := range users {
		usersDTO[i] = userDTOFromDomain(user)
	}

	return usersDTO
}
