package web_fs_repository

import (
	"errors"
	"fmt"
	"io/fs"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

func (r *WebRepository) GetFile(filePath string) ([]byte, error) {
	file, err := fs.ReadFile(r.files, filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf(
				"file: %s: %w",
				filePath,
				core_errors.ErrNotFound,
			)
		}
		return nil, fmt.Errorf(
			"get file: %s: %w",
			filePath,
			err,
		)
	}
	return file, nil
}
