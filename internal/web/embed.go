// Package web embeds the production interface in the Go binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var assets embed.FS

func Assets() fs.FS {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
