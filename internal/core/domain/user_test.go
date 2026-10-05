package domain

import (
	"errors"
	"testing"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func TestUserValidate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{"valid without phone", User{FullName: "Ivan"}, false},
		{"valid with phone", User{FullName: "Ivan", PhoneNumber: ptr("+79998887766")}, false},
		{"empty name", User{FullName: ""}, true},
		{"one letter name", User{FullName: "Я"}, false},
		{"phone without plus", User{FullName: "Ivan", PhoneNumber: ptr("79998887766")}, true},
		{"phone with letters", User{FullName: "Ivan", PhoneNumber: ptr("+7999888abcd")}, true},
		{"short phone", User{FullName: "Ivan", PhoneNumber: ptr("+7999")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, core_errors.ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestUserApplyPatch(t *testing.T) {
	t.Run("null phone clears it", func(t *testing.T) {
		user := User{FullName: "Ivan", PhoneNumber: ptr("+79998887766")}

		if err := user.ApplyPatch(UserPatch{PhoneNumber: setNull[string]()}); err != nil {
			t.Fatal(err)
		}
		if user.PhoneNumber != nil {
			t.Fatalf("phone = %q, want nil", *user.PhoneNumber)
		}
	})

	t.Run("null full name is rejected", func(t *testing.T) {
		user := User{FullName: "Ivan"}

		err := user.ApplyPatch(UserPatch{Fullname: setNull[string]()})
		if !errors.Is(err, core_errors.ErrInvalidArgument) {
			t.Fatalf("err = %v, want ErrInvalidArgument", err)
		}
	})
}

func TestUserApplyPatchExpectedVersion(t *testing.T) {
	user := User{Version: 5, FullName: "Ivan"}

	err := user.ApplyPatch(UserPatch{Fullname: set("Petr"), ExpectedVersion: ptr(4)})
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}
