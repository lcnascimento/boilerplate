package service

import (
	"slices"

	"github.com/lcnascimento/go-kit/errors"
)

type Component string

const (
	API      Component = "api"
	Consumer Component = "consumer"
	Worker   Component = "worker"
	Browser  Component = "browser"
)

type Service struct {
	Name      string
	Processes []Process
}

type Process struct {
	name       string
	components []Component
}

func Daemon(name string, component Component, more ...Component) Process {
	return Process{name: name, components: append([]Component{component}, more...)}
}

func OneShot(name string) Process {
	return Process{name: name}
}

func (p Process) Name() string { return p.name }

func (p Process) Components() []Component { return slices.Clone(p.components) }

func (p Process) LongRunning() bool { return len(p.components) > 0 }

func Validate(services []Service) error {
	serviceNames := map[string]bool{}
	processNames := map[string]bool{}

	for _, svc := range services {
		if svc.Name == "" {
			return errors.New("a service has no name")
		}

		if serviceNames[svc.Name] {
			return errors.New("service %q is declared twice", svc.Name)
		}

		serviceNames[svc.Name] = true

		if len(svc.Processes) == 0 {
			return errors.New("service %q declares no process", svc.Name)
		}

		for _, process := range svc.Processes {
			if err := validateProcess(svc.Name, process, processNames); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateProcess(svc string, process Process, seen map[string]bool) error {
	if process.name == "" {
		return errors.New("service %q has a process with no name", svc)
	}

	if seen[process.name] {
		return errors.New("process %q is declared twice", process.name)
	}

	seen[process.name] = true

	for _, component := range process.components {
		switch component {
		case API, Consumer, Worker, Browser:
		default:
			return errors.New("process %q declares unknown component %q", process.name, component)
		}
	}

	return nil
}
