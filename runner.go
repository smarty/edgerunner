package edgerunner

import (
	"context"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type defaultRunner struct {
	ctx         context.Context
	shutdown    context.CancelFunc
	taskBanner  string
	taskFactory TaskFactory
	waiter      *sync.WaitGroup
	ready       chan bool
	terminate   chan os.Signal
	reload      chan os.Signal
	identifier  int
	logger      Logger
}

func newRunner(config configuration) Runner {
	ctx, shutdown := context.WithCancel(config.Context)

	terminate := make(chan os.Signal, 16)
	reload := make(chan os.Signal, 16)
	signal.Notify(terminate, config.TerminateSignals...)
	signal.Notify(reload, config.ReloadSignals...)

	return &defaultRunner{
		ctx:         ctx,
		shutdown:    shutdown,
		taskBanner:  config.TaskBanner,
		taskFactory: config.TaskFactory,
		waiter:      &sync.WaitGroup{},
		ready:       make(chan bool, 16),
		terminate:   terminate,
		reload:      reload,
		logger:      config.Logger,
	}
}

func (this *defaultRunner) Listen() {
	this.logger.Printf("[INFO] Running configured task [%s]...", this.taskBanner)
	defer this.cleanup()

	this.waiter.Add(1)
	defer this.waiter.Wait()

	go this.awaitTerminate()
	go this.awaitShutdown()
	go this.listen()
}
func (this *defaultRunner) listen() {
	var active ListenCloser

	// ensure that we close the ***CURRENT*** active instance at conclusion of method
	defer func() { closeResource(active) }()

	for {
		active = this.run(active)

		select {
		case <-this.ctx.Done():
			return
		case value := <-this.reload:
			this.logger.Printf("[INFO] Received OS signal [%s], instructing runner to reload configured task...", value)
			continue
		}
	}
}
func (this *defaultRunner) run(active ListenCloser) ListenCloser {
	this.identifier++

	pending := this.taskFactory(this.identifier, this.ready)
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
func (this *defaultRunner) drainChannel() {
	for i := 0; i < len(this.ready); i++ {
		<-this.ready
	}
}
func (this *defaultRunner) awaitTerminate() {
	select {
	case <-this.ctx.Done():
	case value := <-this.terminate:
		this.logger.Printf("[INFO] Received OS signal [%s], shutting down runner along with any associated task(s)...", value)
		this.shutdown()
	}
}
func (this *defaultRunner) awaitShutdown() {
	<-this.ctx.Done()
	this.waiter.Done()
}
func (this *defaultRunner) cleanup() {
	this.shutdown()
	signal.Stop(this.terminate)
	signal.Stop(this.reload)
	close(this.terminate)
	close(this.reload)
	close(this.ready)
	this.logger.Printf("[INFO] The configured runner has completed execution of all specified tasks.")
}

func (this *defaultRunner) Close() error {
	this.logger.Printf("[INFO] Request to close runner received, shutting down runner along with any associated task(s)...")
	this.shutdown()
	return nil
}
func (this *defaultRunner) Reload() {
	if len(this.reload) > 0 {
		return
	}

	select {
	case <-this.ctx.Done(): // shutting down, don't reload
	case this.reload <- syscall.Signal(0):
	default: // buffer full, don't send more
	}
}

func closeResource(resource io.Closer) {
	if resource != nil {
		_ = resource.Close()
	}
}
