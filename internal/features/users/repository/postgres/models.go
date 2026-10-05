package users_postgres_repository

import "github.com/dadqeds/todoapp/internal/core/domain"

const userColumns = `id, version, full_name, phone_number, telegram_id, timezone, remind_enabled, digest_enabled, digest_minute`

type UserModel struct {
	ID          int
	Version     int
	FullName    string
	PhoneNumber *string
	TelegramID  *int64
	Timezone    string

	RemindEnabled bool
	DigestEnabled bool
	DigestMinute  int
}

// scanner покрывает и core_postgres_pool.Row, и core_postgres_pool.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanUserModel(s scanner) (UserModel, error) {
	var userModel UserModel

	err := s.Scan(
		&userModel.ID,
		&userModel.Version,
		&userModel.FullName,
		&userModel.PhoneNumber,
		&userModel.TelegramID,
		&userModel.Timezone,
		&userModel.RemindEnabled,
		&userModel.DigestEnabled,
		&userModel.DigestMinute,
	)

	return userModel, err
}

func userDomainFromModel(user UserModel) domain.User {
	u := domain.NewUser(
		user.ID,
		user.Version,
		user.FullName,
		user.PhoneNumber,
		user.TelegramID,
	)
	u.Timezone = user.Timezone
	u.RemindEnabled = user.RemindEnabled
	u.DigestEnabled = user.DigestEnabled
	u.DigestMinute = user.DigestMinute
	return u
}

func userDomainsFromModels(users []UserModel) []domain.User {
	usersDomains := make([]domain.User, len(users))

	for i, user := range users {
		usersDomains[i] = userDomainFromModel(user)
	}

	return usersDomains
}
