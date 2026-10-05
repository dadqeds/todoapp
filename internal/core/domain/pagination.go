package domain

import (
	"fmt"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

const (
	DefaultPageLimit = 50
	MaxPageLimit     = 500
)

// NormalizePagination подставляет дефолты и проверяет границы limit/offset,
// чтобы запрос без limit не выгружал всю таблицу.
func NormalizePagination(limit *int, offset *int) (int, int, error) {
	l, o := DefaultPageLimit, 0

	if limit != nil {
		if *limit < 0 || *limit > MaxPageLimit {
			return 0, 0, fmt.Errorf(
				"limit must be between 0 and %d: %w",
				MaxPageLimit,
				core_errors.ErrInvalidArgument,
			)
		}
		l = *limit
	}

	if offset != nil {
		if *offset < 0 {
			return 0, 0, fmt.Errorf(
				"offset must be non-negative: %w",
				core_errors.ErrInvalidArgument,
			)
		}
		o = *offset
	}

	return l, o, nil
}
