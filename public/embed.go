// Package public встраивает статические файлы фронтенда в бинарник,
// чтобы UI работал независимо от рабочей директории и в Docker.
package public

import "embed"

//go:embed index.html assets
var FS embed.FS
