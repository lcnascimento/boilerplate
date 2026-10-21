package main

import (
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

var services = []service.Service{
	{
		Name:      "web",
		Runtime:   service.Node,
		Processes: []service.Process{service.Daemon("web", service.API, service.Browser)},
	},
	{
		Name:      "iam",
		Processes: []service.Process{service.OneShot("iam-migrate")},
	},
	// example:begin
	{
		Name: "notes",
		Processes: []service.Process{
			service.Daemon("notes-api-frontend", service.API).WithDBPool(),
			service.Daemon("notes-worker-forwarder", service.Worker).WithDBPool(),
			service.OneShot("notes-migrate"),
			service.OneShot("notes-schemas"),
		},
	},
	{
		Name: "audit",
		Processes: []service.Process{
			service.Daemon("audit-consumer", service.Consumer).WithConsumerGroup("audit").WithDBPool(),
			service.OneShot("audit-migrate"),
		},
	},
	// example:end
}
