package main

import (
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

var services = []service.Service{
	// example:begin
	{
		Name: "notes",
		Processes: []service.Process{
			service.Daemon("notes-api-frontend", service.API),
			service.Daemon("notes-worker-forwarder", service.Worker),
			service.OneShot("notes-migrate"),
			service.OneShot("notes-schemas"),
		},
	},
	{
		Name: "audit",
		Processes: []service.Process{
			service.Daemon("audit-consumer", service.Consumer),
			service.OneShot("audit-migrate"),
		},
	},
	// example:end
}
