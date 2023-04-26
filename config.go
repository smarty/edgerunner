package edgerunner

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func New(options ...option) Runner {
	var config configuration
	Options.apply(options...)(&config)
	return newRunner(config)
}

func (singleton) Context(value context.Context) option {
	return func(this *configuration) { this.context, this.cancel = context.WithCancel(value) }
}
func (singleton) WatchTerminateSignals(values ...os.Signal) option {
	return func(this *configuration) {
		this.terminationSignals = make(chan os.Signal, 16)
		signal.Notify(this.terminationSignals, values...)
	}
}
func (singleton) WatchReloadSignals(values ...os.Signal) option {
	return func(this *configuration) {
		this.reloadSignals = make(chan os.Signal, 16)
		signal.Notify(this.reloadSignals, values...)
	}
}
func (singleton) TaskName(value string) option {
	return func(this *configuration) { this.taskName = value }
}
func (singleton) TaskVersion(value string) option {
	return func(this *configuration) { this.taskVersion = value }
}
func (singleton) TaskFactory(value TaskFactory) option {
	return func(this *configuration) { this.taskFactory = value }
}
func (singleton) Logger(value Logger) option {
	return func(this *configuration) { this.log = value }
}

func (singleton) apply(options ...option) option {
	return func(this *configuration) {
		for _, item := range Options.defaults(options...) {
			item(this)
		}
	}
}
func (singleton) defaults(options ...option) []option {
	return append([]option{
		Options.Context(context.Background()),
		Options.WatchTerminateSignals(syscall.SIGINT, syscall.SIGTERM),
		Options.WatchReloadSignals(syscall.SIGHUP),
		Options.TaskName("unknown"),
		Options.TaskVersion("unknown"),
		Options.TaskFactory(func(id int, ready chan<- bool) Task { return nil }),
		Options.Logger(log.Default()),
	}, options...)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type configuration struct {
	log                Logger
	context            context.Context
	cancel             context.CancelFunc
	reloadSignals      chan os.Signal
	terminationSignals chan os.Signal
	taskName           string
	taskVersion        string
	taskFactory        TaskFactory
}

type option func(*configuration)
type singleton struct{}

var Options singleton
