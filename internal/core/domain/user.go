package domain

import (
	"fmt"
	"regexp"
	"time"

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

	// Timezone — пояс IANA (Europe/Moscow), по нему считаются повторы и время
	// уведомлений. Пустой — UTC.
	Timezone string

	// Уведомления от бота: напоминания о сроках и утренняя сводка.
	// DigestMinute — время сводки, минуты от полуночи по местному времени.
	RemindEnabled bool
	DigestEnabled bool
	DigestMinute  int
}

const DefaultDigestMinute = 9 * 60

// Location возвращает часовой пояс пользователя, при ошибке — UTC.
func (u *User) Location() *time.Location {
	if u.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
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

	if u.DigestMinute < 0 || u.DigestMinute > 24*60-1 {
		return fmt.Errorf(
			"invalid `DigestMinute` %d: %w",
			u.DigestMinute,
			core_errors.ErrInvalidArgument,
		)
	}

	if u.Timezone != "" {
		if _, err := time.LoadLocation(u.Timezone); err != nil || len(u.Timezone) > 64 {
			return fmt.Errorf(
				"invalid `Timezone` %q: %w",
				u.Timezone,
				core_errors.ErrInvalidArgument,
			)
		}
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
	Timezone    Nullable[string]

	RemindEnabled Nullable[bool]
	DigestEnabled Nullable[bool]
	DigestMinute  Nullable[int]

	// ExpectedVersion — см. TaskPatch.ExpectedVersion.
	ExpectedVersion *int
}

func (p *UserPatch) Validate() error {
	if (p.RemindEnabled.Set && p.RemindEnabled.Value == nil) ||
		(p.DigestEnabled.Set && p.DigestEnabled.Value == nil) ||
		(p.DigestMinute.Set && p.DigestMinute.Value == nil) {
		return fmt.Errorf(
			"notification settings can't be patched to NULL: %w",
			core_errors.ErrInvalidArgument,
		)
	}

	if p.Timezone.Set && p.Timezone.Value == nil {
		return fmt.Errorf(
			"`Timezone` can't be patched to NULL: %w",
			core_errors.ErrInvalidArgument,
		)
	}

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

	if patch.Timezone.Set {
		tmp.Timezone = *patch.Timezone.Value
	}

	if patch.RemindEnabled.Set {
		tmp.RemindEnabled = *patch.RemindEnabled.Value
	}

	if patch.DigestEnabled.Set {
		tmp.DigestEnabled = *patch.DigestEnabled.Value
	}

	if patch.DigestMinute.Set {
		tmp.DigestMinute = *patch.DigestMinute.Value
	}

	if err := tmp.Validate(); err != nil {
		return fmt.Errorf("validate patched user: %w", err)
	}

	*u = tmp
	return nil
}
