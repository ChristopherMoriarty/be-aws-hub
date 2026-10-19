package migrations

import "embed"

// FS contains the goose SQL migrations.
//
//go:embed *.sql
var FS embed.FS
