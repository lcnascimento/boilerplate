package service

import (
	"slices"
	"strings"

	"github.com/lcnascimento/go-kit/errors"
)

type Component string

const (
	API      Component = "api"
	Consumer Component = "consumer"
	Worker   Component = "worker"
	Browser  Component = "browser"
)

var ErrActorAuthWithoutAPI = errors.New("process declares actor auth but serves no api")

var components = []Component{API, Consumer, Worker, Browser}

type Runtime int

const (
	Go Runtime = iota
	Node
)

const DefaultSeriesBudget = 2000

type Service struct {
	Name         string
	Runtime      Runtime
	Processes    []Process
	SeriesBudget int
	SLOs         []SLO
}

func (s *Service) EffectiveSeriesBudget() int {
	if s.SeriesBudget == 0 {
		return DefaultSeriesBudget
	}

	return s.SeriesBudget
}

type Severity string

const (
	Critical Severity = "critical"
	Warning  Severity = "warning"
)

type LogAlert struct {
	Name            string
	Pipeline        string
	MaxRecordsPer5m int
	Severity        Severity
	Summary         string
	Description     string
}

type Process struct {
	name       string
	components []Component
	options    *processOptions
}

type processOptions struct {
	dbPool        bool
	actorAuth     bool
	consumerGroup string
	logAlerts     []LogAlert
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

func (p Process) WithDBPool() Process {
	return p.with(func(o *processOptions) { o.dbPool = true })
}

func (p Process) DBPool() bool { return p.opts().dbPool }

func (p Process) WithActorAuth() Process {
	return p.with(func(o *processOptions) { o.actorAuth = true })
}

func (p Process) ActorAuth() bool { return p.opts().actorAuth }

func (p Process) WithConsumerGroup(group string) Process {
	return p.with(func(o *processOptions) { o.consumerGroup = group })
}

func (p Process) ConsumerGroup() string { return p.opts().consumerGroup }

func (p Process) WithLogAlert(alert *LogAlert) Process {
	return p.with(func(o *processOptions) { o.logAlerts = append(o.logAlerts, *alert) })
}

func (p Process) LogAlerts() []LogAlert { return slices.Clone(p.opts().logAlerts) }

func (p Process) Runs(component Component) bool { return slices.Contains(p.components, component) }

func (p Process) opts() processOptions {
	if p.options == nil {
		return processOptions{}
	}

	return *p.options
}

func (p Process) with(change func(*processOptions)) Process {
	options := p.opts()
	options.logAlerts = slices.Clone(options.logAlerts)
	change(&options)
	p.options = &options

	return p
}

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

		if svc.SeriesBudget < 0 {
			return errors.New("service %q declares a negative series budget", svc.Name)
		}

		if len(svc.Processes) == 0 {
			return errors.New("service %q declares no process", svc.Name)
		}

		for _, process := range svc.Processes {
			if err := validateProcess(svc.Name, process, processNames); err != nil {
				return err
			}

			if err := validateRuntime(svc.Runtime, process); err != nil {
				return err
			}
		}

		if err := validateLogAlerts(&svc); err != nil {
			return err
		}

		if err := validateSLOs(&svc); err != nil {
			return err
		}
	}

	return nil
}

func validateLogAlerts(svc *Service) error {
	names := map[string]bool{}

	for _, process := range svc.Processes {
		alerts := process.LogAlerts()
		for i := range alerts {
			alert := &alerts[i]

			switch {
			case !process.LongRunning():
				return errors.New("one-shot process %q cannot declare log alert %s", process.name, alert.Name)
			case alert.Name == "" || alert.Pipeline == "":
				return errors.New("service %q has a log alert with no name or pipeline", svc.Name)
			case names[alert.Name]:
				return errors.New("service %q declares log alert %s twice", svc.Name, alert.Name)
			case alert.MaxRecordsPer5m < 0:
				return errors.New("log alert %s has a negative threshold", alert.Name)
			}

			names[alert.Name] = true
		}
	}

	return nil
}

type runtimeTraits struct {
	name       string
	components []Component
	dbPool     bool
	goKit      bool
}

var runtimes = map[Runtime]runtimeTraits{
	Go:   {name: "Go", components: components, dbPool: true, goKit: true},
	Node: {name: "Node", components: []Component{API, Browser}},
}

func (r Runtime) RunsGoKit() bool { return runtimes[r].goKit }

func listed(components []Component) string {
	names := make([]string, len(components))
	for i, c := range components {
		names[i] = string(c)
	}

	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

func validateRuntime(runtime Runtime, process Process) error {
	traits, ok := runtimes[runtime]
	if !ok {
		return errors.New("process %q declares unknown runtime %d", process.name, runtime)
	}

	uninstrumented := func(c Component) bool { return !slices.Contains(traits.components, c) }
	if (process.DBPool() && !traits.dbPool) || slices.ContainsFunc(process.components, uninstrumented) {
		return errors.New("process %q runs on %s, which only instruments %s", process.name, traits.name,
			listed(traits.components))
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
		if !slices.Contains(components, component) {
			return errors.New("process %q declares unknown component %q", process.name, component)
		}
	}

	if process.Runs(Consumer) != (process.ConsumerGroup() != "") {
		return errors.New("process %q must declare a consumer group if, and only if, it runs a consumer", process.name)
	}

	if process.ActorAuth() && !process.Runs(API) {
		return ErrActorAuthWithoutAPI.WithAttribute("process", process.name)
	}

	if process.DBPool() && !process.LongRunning() {
		return errors.New("one-shot process %q cannot declare a database pool", process.name)
	}

	return nil
}
