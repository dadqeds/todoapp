package domain

import (
	"fmt"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func checkExpectedVersion(expected *int, actual int) error {
	if expected != nil && *expected != actual {
		return fmt.Errorf(
			"version mismatch: expected %d, actual %d: %w",
			*expected,
			actual,
			core_errors.ErrConflict,
		)
	}

	return nil
}
