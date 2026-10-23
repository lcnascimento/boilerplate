package events

import (
	"embed"

	"github.com/lcnascimento/boilerplate/backend/services/iam/pkg/eventbus"
)

//go:embed proto
var embedded embed.FS

var Protos = eventbus.MustProtos(embedded)
