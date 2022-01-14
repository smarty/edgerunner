package edgerunner

import (
	"context"
	"os"
	"os/signal"
)

type signalWatcher struct {
	parent   context.Context
	callback func()
	watching []os.Signal
	notify   chan os.Signal
}

func newSignalWatcher(parent context.Context, callback func(), watching []os.Signal) ListenCloser {
	return &signalWatcher{
		parent:   parent,
		callback: callback,
		watching: watching,
		notify:   make(chan os.Signal, 16),
	}
}

func (this *signalWatcher) Listen() {
	defer close(this.notify)
	signal.Notify(this.notify, this.watching...)
	defer signal.Stop(this.notify)

	select {
	case <-this.parent.Done():
		break
	case <-this.notify:
		this.drainSignals()
		this.callback()
	}
}
func (this *signalWatcher) drainSignals() {
	for i := 0; i < len(this.notify); i++ {
		<-this.notify
	}
}

func (this *signalWatcher) Close() error { return nil }
