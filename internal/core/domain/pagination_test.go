package domain

import (
	"errors"
	"testing"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name               string
		limit, offset      *int
		wantLimit, wantOff int
		wantErr            bool
	}{
		{name: "defaults", wantLimit: DefaultPageLimit},
		{name: "explicit", limit: ptr(10), offset: ptr(20), wantLimit: 10, wantOff: 20},
		{name: "max limit", limit: ptr(MaxPageLimit), wantLimit: MaxPageLimit},
		{name: "limit above max", limit: ptr(MaxPageLimit + 1), wantErr: true},
		{name: "negative limit", limit: ptr(-1), wantErr: true},
		{name: "negative offset", offset: ptr(-1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, o, err := NormalizePagination(tt.limit, tt.offset)
			if tt.wantErr {
				if !errors.Is(err, core_errors.ErrInvalidArgument) {
					t.Fatalf("err = %v, want ErrInvalidArgument", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if l != tt.wantLimit || o != tt.wantOff {
				t.Fatalf("got (%d, %d), want (%d, %d)", l, o, tt.wantLimit, tt.wantOff)
			}
		})
	}
}
