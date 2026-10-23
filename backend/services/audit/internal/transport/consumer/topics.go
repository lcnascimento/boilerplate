package consumer

import (
	"slices"

	iamevents "github.com/lcnascimento/boilerplate/backend/services/iam/pkg/events"
)

var Topics = slices.Concat(
	iamevents.Topics,
	// example:begin
	notesTopics,
	// example:end
)
