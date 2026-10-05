package users_transport_http

import "github.com/dadqeds/todoapp/internal/core/domain"

type UserDTOResponse struct {
	ID          int     `json:"id"            example:"10"`
	Version     int     `json:"version"       example:"3"`
	FullName    string  `json:"full_name"     example:"Ivan Ivanov"`
	PhoneNumber *string `json:"phone_number"  example:"+79998887766"`
	TelegramID  *int64  `json:"telegram_id"   example:"123456789"`
	Timezone    string  `json:"timezone"      example:"Europe/Moscow"`
}

func userDTOFromDomain(user domain.User) UserDTOResponse {
	return UserDTOResponse{
		ID:          user.ID,
		Version:     user.Version,
		FullName:    user.FullName,
		PhoneNumber: user.PhoneNumber,
		TelegramID:  user.TelegramID,
		Timezone:    user.Timezone,
	}
}

func usersDTOFromDomains(users []domain.User) []UserDTOResponse {
	usersDTO := make([]UserDTOResponse, len(users))

	for i, user := range users {
		usersDTO[i] = userDTOFromDomain(user)
	}

	return usersDTO
}
