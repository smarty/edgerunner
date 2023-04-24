package edgerunner

import (
	"context"
	"io"
	"os"
	"os/signal"
	"sync"
)

type defaultRunner struct {
	logger       Logger
	ctx          context.Context
	cancel       context.CancelFunc
	name         string
	version      string
	factory      TaskFactory
	ready        chan bool
	reloads      chan os.Signal
	terminations chan os.Signal
	waiters      chan func()
	counter      int
	running      Task
}

func newRunner(config configuration) Runner {
	ctx, cancel := context.WithCancel(config.Context)
	reloads := make(chan os.Signal, 16)
	terminations := make(chan os.Signal, 16)
	signal.Notify(reloads, config.ReloadSignals...)
	signal.Notify(terminations, config.TerminateSignals...)
	return &defaultRunner{
		logger:       config.Logger,
		ctx:          ctx,
		cancel:       cancel,
		name:         config.TaskName,
		version:      config.TaskVersion,
		factory:      config.TaskFactory,
		ready:        make(chan bool, 16),
		reloads:      reloads,
		terminations: terminations,
		waiters:      make(chan func()),
	}
}

func (this *defaultRunner) Close() error {
	this.logger.Printf("[INFO] Request to close runner received, shutting down runner along with any associated task(s)...")
	this.cancel()
	return nil
}
func (this *defaultRunner) Reload() {
	panic("NOT YET IMPLEMENTED") // TODO
}
func (this *defaultRunner) Listen() {
	this.logger.Printf("[INFO] Running configured task [%s] at version [%s]...", this.name, this.version)

	go this.listenAll()

	for wait := range this.waiters {
		wait()
	}

	this.logger.Printf("[INFO] The configured runner has completed execution of all specified tasks.")
}
func (this *defaultRunner) listenAll() {
	defer close(this.waiters)
	for {
		next := this.prepareNextTask()
		if next != nil {
			this.waiters <- startNextTask(this.ctx, this.running, next, this.ready)
			this.running = next
		}

		select {
		case <-this.reloads:
			continue
		case <-this.terminations:
			return
		case <-this.ctx.Done():
			return
		}
	}
}
func (this *defaultRunner) prepareNextTask() Task {
	this.counter++

	next := this.factory(this.counter, this.ready)
	if next == nil {
		this.logger.Printf("[WARN] No task created for ID [%d].", this.counter)
		return nil
	}

	err := next.Initialize(this.ctx)
	if err != nil {
		this.logger.Printf("[WARN] Unable to initialize task [%d]: %s", this.counter, err)
		closeResource(next)
		return nil
	}

	return next
}
func startNextTask(ctx context.Context, previous, this Task, ready chan bool) (wait func()) {
	waiter := &sync.WaitGroup{}

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		this.Listen()
	}()

	waiter.Add(1)
	go func() {
		defer waiter.Done()
		defer closeResource(this)
		select {
		case <-ctx.Done():
		case isReady := <-ready:
			if isReady {
				closeResource(previous)
			}
		}
	}()

	return waiter.Wait
}
func closeResource(resource io.Closer) {
	if resource != nil {
		_ = resource.Close()
	}
}
