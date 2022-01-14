package edgerunner

import (
	"context"
	"os"
	"syscall"
)

func New(options ...option) TaskRunner {
	var config configuration
	Options.apply(options...)(&config)
	return newConcurrentRunner(config)
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
func (singleton) ConcurrentTask(value ConcurrentTaskFactory) option {
	return func(this *configuration) { this.ConcurrentTask = value }
}
func (singleton) Monitor(value Monitor) option {
	return func(this *configuration) { this.Monitor = value }
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
		Options.ConcurrentTask(noop.ConcurrentFactory),
		Options.Monitor(noop),
		Options.Logger(noop),
	}, options...)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type configuration struct {
	Context          context.Context
	TerminateSignals []os.Signal
	ReloadSignals    []os.Signal
	ConcurrentTask   ConcurrentTaskFactory
	Monitor          Monitor
	Logger           Logger
}
type option func(*configuration)
type singleton struct{}

var Options singleton

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type nop struct{}

func (this *nop) ConcurrentFactory(int, chan<- bool) Task { return this }

func (*nop) Initialize(context.Context) error { return nil }
func (*nop) Listen()                          {}
func (*nop) Close() error                     { return nil }
func (*nop) Printf(string, ...interface{})    {}
