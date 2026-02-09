package web

import "embed"

// DistFS contains the built web UI assets from the dist directory.
//
//go:embed all:dist
var DistFS embed.FS
