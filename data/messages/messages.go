package messages

import "embed"

//go:embed active.en.toml
//go:embed active.ru.toml
var FS embed.FS
