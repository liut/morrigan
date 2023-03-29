package schemas

import (
	"embed"
	"io/fs"
)

//go:embed pg_??_*_*.sql
var dbfs embed.FS

func SchemaFS() fs.FS {
	return &dbfs
}

//go:embed 20????????????_*.up.sql
var upgrades embed.FS

func UpgradesFS() fs.FS {
	return upgrades
}
