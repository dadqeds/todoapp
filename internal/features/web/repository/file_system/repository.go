package web_fs_repository

import "io/fs"

type WebRepository struct {
	files fs.FS
}

func NewWebRepository(files fs.FS) *WebRepository {
	return &WebRepository{
		files: files,
	}
}
