package edgerunner

import (
	"context"
	"os"
	"strings"
	"syscall"
)

func New(options ...option) Runner {
	var config configuration
	Options.apply(options...)(&config)
	return newRunner(config)
}

func (singleton) Context(value context.Context) option {
	return func(this *configuration) { this.Context = value }
}
func (singleton) WatchTerminateSignals(values ...os.Signal) option {
	return func(this *configuration) { this.TerminateSignals = values }
}
func (singleton) WatchReloadSignals(values ...os.Signal) option {
	return func(this *configuration) { this.ReloadSignals = values }
}
func (singleton) Task(name, version string, value TaskFactory) option {
	return func(this *configuration) {
		name = strings.TrimSpace(name)
		version = strings.TrimSpace(version)

		if len(name) == 0 {
			name = "(unknown)"
		}
		if len(version) > 0 {
			version = " " + version
		}
		this.TaskBanner = name + version
		this.TaskFactory = value
	}
}
func (singleton) Logger(value Logger) option {
	return func(this *configuration) { this.Logger = value }
}

func (singleton) apply(options ...option) option {
	return func(this *configuration) {
		for _, item := range Options.defaults(options...) {
			item(this)
		}
	}
}
func (singleton) defaults(options ...option) []option {
	noop := &nop{}
	return append([]option{
		Options.Context(context.Background()),
		Options.WatchTerminateSignals(syscall.SIGINT, syscall.SIGTERM),
		Options.WatchReloadSignals(syscall.SIGHUP),
		Options.Task("", "", noop.Factory),
		Options.Logger(noop),
	}, options...)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type configuration struct {
	Context          context.Context
	TerminateSignals []os.Signal
	ReloadSignals    []os.Signal
	TaskBanner       string
	TaskFactory      TaskFactory
	Logger           Logger
}
type option func(*configuration)
type singleton struct{}

var Options singleton

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type nop struct{}

func (this *nop) Factory(int, chan<- bool) Task { return this }

func (*nop) Initialize(context.Context) error { return nil }
func (*nop) Listen()                          {}
func (*nop) Close() error                     { return nil }
func (*nop) Printf(string, ...interface{})    {}
