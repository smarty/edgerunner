package edgerunner

import (
	"context"
	"io"
	"sync"
)

type concurrentRunner struct {
	ctx              context.Context
	shutdown         context.CancelFunc
	newTask          ConcurrentTaskFactory
	terminateWatcher ListenCloser
	reloadWatcher    ListenCloser
	waiter           *sync.WaitGroup
	ready            chan bool
	reload           chan struct{}
	identifier       int
	logger           Logger
}

func newConcurrentRunner(config configuration) TaskRunner {
	ctx, shutdown := context.WithCancel(config.Context)
	this := &concurrentRunner{
		ctx:              ctx,
		shutdown:         shutdown,
		newTask:          config.ConcurrentTask,
		terminateWatcher: newSignalWatcher(ctx, shutdown, config.TerminateSignals),
		waiter:           &sync.WaitGroup{},
		ready:            make(chan bool, 16),
		reload:           make(chan struct{}, 16),
		logger:           config.Logger,
	}

	this.reloadWatcher = newSignalWatcher(this.ctx, this.Reload, config.TerminateSignals)
	return this
}

func (this *concurrentRunner) Listen() {
	defer this.cleanup()

	this.waiter.Add(1)
	defer this.waiter.Wait()

	go this.terminateWatcher.Listen()
	go this.reloadWatcher.Listen()
	go this.awaitShutdown()
	go this.listen()
}
func (this *concurrentRunner) listen() {
	var active ListenCloser

	// ensure that we close the ***CURRENT*** active instance at conclusion of method
	defer func() { closeResource(active) }()

	for {
		active = this.run(active)

		select {
		case <-this.ctx.Done():
			return
		case <-this.reload:
			continue
		}
	}
}
func (this *concurrentRunner) run(active ListenCloser) ListenCloser {
	this.identifier++

	pending := this.newTask(this.identifier, this.ready)
	if pending == nil {
		this.logger.Printf("[WARN] No task created for ID [%d].", this.identifier)
		return active
	} else if err := pending.Initialize(this.ctx); err != nil {
		this.logger.Printf("[WARN] Unable to initialize task [%d]: %s", this.identifier, err)
		closeResource(pending)
		return active
	}

	this.waiter.Add(1)
	go func() {
		defer this.waiter.Done()
		pending.Listen()
	}()

	select {
	case <-this.ctx.Done():
		closeResource(pending)
		return active
	case readyState := <-this.ready:
		this.drainChannel()
		if readyState {
			this.logger.Printf("[INFO] Pending task [%d] has arrived at a ready state; shutting down previous task, if any.", this.identifier)
			closeResource(active)
			return pending
		} else {
			this.logger.Printf("[WARN] Pending task [%d] did not arrive at a ready state; continuing with previous task, if any.", this.identifier)
			closeResource(pending)
			return active
		}
	}
}

func (this *concurrentRunner) drainChannel() {
	for i := 0; i < len(this.ready); i++ {
		<-this.ready // drain ready channel; we only want the first value
	}
}
func (this *concurrentRunner) awaitShutdown() {
	<-this.ctx.Done()
	this.waiter.Done()
}
func (this *concurrentRunner) cleanup() {
	this.shutdown()
	closeResource(this.terminateWatcher)
	closeResource(this.reloadWatcher)
	close(this.reload)
	close(this.ready)
}

func (this *concurrentRunner) Close() error {
	this.shutdown()
	return nil
}
func (this *concurrentRunner) Reload() {
	select {
	case this.reload <- struct{}{}: // reload message sent
	default: // buffer full, don't send more
	}
}

func closeResource(resource io.Closer) {
	if resource != nil {
		_ = resource.Close()
	}
}
