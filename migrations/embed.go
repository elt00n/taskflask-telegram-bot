// Package migrations предоставляет приложению SQL-миграции, встроенные в бинарный файл.
package migrations

import "embed"

// Files содержит все миграции с расширением .up.sql.
//
//go:embed *.up.sql
var Files embed.FS
