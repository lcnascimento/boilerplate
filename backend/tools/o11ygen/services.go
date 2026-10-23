package main

import (
	"github.com/lcnascimento/boilerplate/backend/tools/o11ygen/internal/service"
)

var services = []service.Service{
	{
		Name:      "web",
		Runtime:   service.Node,
		Processes: []service.Process{service.Daemon("web", service.API, service.Browser).WithActorAuth()},
	},
	{
		Name: "iam",
		Processes: []service.Process{
			service.Daemon("iam-worker", service.Worker).WithDBPool().WithIdentityChanges(),
			service.OneShot("iam-migrate"),
			service.OneShot("iam-schemas"),
		},
	},
	{
		Name: "audit",
		Processes: []service.Process{
			service.Daemon("audit-consumer", service.Consumer).WithConsumerGroup("audit").WithDBPool().WithAuditLog(),
			service.OneShot("audit-migrate"),
		},
	},
	// example:begin
	{
		Name: "notes",
		Processes: []service.Process{
			service.Daemon("notes-api-frontend", service.API).WithDBPool().WithActorAuth(),
			service.Daemon("notes-worker-forwarder", service.Worker).WithDBPool(),
			service.Daemon("notes-consumer", service.Consumer).WithConsumerGroup("notes").WithDBPool(),
			service.OneShot("notes-migrate"),
			service.OneShot("notes-schemas"),
		},
	},
	// example:end
}
