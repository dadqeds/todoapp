package domain

import (
	"fmt"
	"regexp"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

var phoneNumberRegexp = regexp.MustCompile(`^\+[0-9]+$`)

const (
	FullNameMinLen = 1
	FullNameMaxLen = 100
)

type User struct {
	ID          int
	Version     int
	FullName    string
	PhoneNumber *string

	// TelegramID — id аккаунта Telegram, через который пользователь входит.
	// nil у пользователей, созданных до входа через Telegram.
	TelegramID *int64
}

func NewUser(
	id int,
	version int,
	fullName string,
	phoneNumber *string,
	telegramID *int64,
) User {
	return User{
		ID:          id,
		Version:     version,
		FullName:    fullName,
		PhoneNumber: phoneNumber,
		TelegramID:  telegramID,
	}
}

func NewUserUninitialized(
	fullName string,
	phoneNumber *string,
) User {
	return NewUser(
		UninitializedID,
		UninitializedVersion,
		fullName,
		phoneNumber,
		nil,
	)
}

// NewTelegramUserUninitialized создаёт пользователя при первом входе через Telegram.
func NewTelegramUserUninitialized(fullName string, telegramID int64) User {
	return NewUser(
		UninitializedID,
		UninitializedVersion,
		fullName,
		nil,
		&telegramID,
	)
}

func (u *User) Validate() error {
	fullnameLen := len([]rune(u.FullName))

	if fullnameLen < FullNameMinLen || fullnameLen > FullNameMaxLen {
		return fmt.Errorf(
			"invalid `FullName` len: %d: %w",
			fullnameLen,
			core_errors.ErrInvalidArgument,
		)
	}

	if u.PhoneNumber != nil {
		phoneNumberLen := len([]rune(*u.PhoneNumber))
		if phoneNumberLen < 10 || phoneNumberLen > 15 {
			return fmt.Errorf(
				"invalid `PhoneNumber` len: %d: %w",
				phoneNumberLen,
				core_errors.ErrInvalidArgument,
			)
		}
		if !phoneNumberRegexp.MatchString(*u.PhoneNumber) {
			return fmt.Errorf(
				"invalid `PhoneNumber` format: %w",
				core_errors.ErrInvalidArgument,
			)
		}
	}
	return nil
}

type UserPatch struct {
	Fullname    Nullable[string]
	PhoneNumber Nullable[string]

	// ExpectedVersion — см. TaskPatch.ExpectedVersion.
	ExpectedVersion *int
}

func NewUserPatch(
	fullName Nullable[string],
	phoneNumber Nullable[string],
	expectedVersion *int,
) UserPatch {
	return UserPatch{
		Fullname:        fullName,
		PhoneNumber:     phoneNumber,
		ExpectedVersion: expectedVersion,
	}
}

func (p *UserPatch) Validate() error {
	if p.Fullname.Set && p.Fullname.Value == nil {
		return fmt.Errorf(
			"`Fullname` can't be patched to NULL: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	return nil
}

func (u *User) ApplyPatch(patch UserPatch) error {
	if err := patch.Validate(); err != nil {
		return fmt.Errorf("validate user patch: %w", err)
	}

	if err := checkExpectedVersion(patch.ExpectedVersion, u.Version); err != nil {
		return err
	}

	tmp := *u

	if patch.Fullname.Set {
		tmp.FullName = *patch.Fullname.Value
	}

	if patch.PhoneNumber.Set {
		tmp.PhoneNumber = patch.PhoneNumber.Value
	}

	if err := tmp.Validate(); err != nil {
		return fmt.Errorf("validate patched user: %w", err)
	}

	*u = tmp
	return nil
}
