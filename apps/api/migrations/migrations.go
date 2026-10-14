// Package migrations embeds the ordered SQL migrations applied at startup.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
