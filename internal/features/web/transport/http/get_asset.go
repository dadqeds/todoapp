package web_transport_http

import (
	"io/fs"
	"net/http"
	"strings"
)

// newAssetsHandler отдаёт файлы из каталога assets/ без листинга директорий.
func newAssetsHandler(files fs.FS) http.HandlerFunc {
	fileServer := http.FileServerFS(files)

	return func(rw http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(rw, r)
			return
		}

		// У встроенных файлов нет времени изменения, поэтому браузер
		// должен перепроверять их при каждой загрузке страницы.
		rw.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(rw, r)
	}
}
