package htdocs

import (
	"embed"
	"io/fs"
)

//go:embed all:?*
var static embed.FS

func FS() fs.FS {
	return &static
}
