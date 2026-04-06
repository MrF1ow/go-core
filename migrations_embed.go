package core

import "embed"

//go:embed migrations/*.sql
var coreMigrationsFS embed.FS
