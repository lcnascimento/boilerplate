package events

import (
	"embed"
	"io/fs"
)

//go:embed proto
var embedded embed.FS

var Protos, _ = fs.Sub(embedded, "proto")
