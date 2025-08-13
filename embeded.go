package ucmsv2

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

//go:embed locales/*.toml
var Locales embed.FS
