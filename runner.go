package edgerunner

import (
	"io"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

type defaultRunner struct {
	config
	id   *atomic.Int32
	task Task
}

func newRunner(config config) Runner {
	return &defaultRunner{
		config: config,
		id:     new(atomic.Int32),
	}
}
func (this *defaultRunner) Close() error {
	this.log.Printf("[INFO] Request to close runner received, shutting down runner along with any associated task(s)...")
	this.cancel()
	return nil
}
func (this *defaultRunner) Reload() {
	select {
	case <-this.context.Done(): // already shut down
	case this.reloads <- syscall.Signal(0):
	default: // reloads chan may be full
	}
}
func (this *defaultRunner) Listen() {
	this.log.Printf("[INFO] Running configured task [%s] at version [%s]...", this.name, this.version)
	this.listen()
	signal.Stop(this.terminations)
	signal.Stop(this.reloads)
	this.log.Printf("[INFO] The configured runner has completed execution of all specified tasks.")
}

func (this *defaultRunner) listen() {
	var final sync.WaitGroup
	tasks := make(chan func())
	go this.listenAll(tasks)
	for task := range tasks {
		if task != nil {
			final.Add(1)
			go act(task, final.Done)
		}
	}
	final.Wait()
}
func (this *defaultRunner) listenAll(waiters chan func()) {
	defer close(waiters)
	for {
		waiters <- this.startNextTask(this.task)

		select {
		case value := <-this.reloads:
			this.log.Printf("[INFO] Received OS reload signal [%v], instructing runner to reload configured task...", value)
			continue
		case value := <-this.terminations:
			this.log.Printf("[INFO] Received OS terminate signal [%v], shutting down runner along with any associated task(s)...", value)
			this.cancel()
			return
		case <-this.context.Done():
			return
		}
	}
}
func (this *defaultRunner) startNextTask(older Task) (wait func()) {
	id := int(this.id.Add(1))
	ready := make(chan bool, 16)
	newer := this.factory(id, ready)
	if newer == nil {
		this.log.Printf("[WARN] No task created for ID [%d].", id)
		return nil
	}

	err := newer.Initialize(this.context)
	if err != nil {
		this.log.Printf("[WARN] Unable to initialize task [%d]: %s", id, err)
		closeResource(newer)
		return nil
	}

	this.task = newer

	return awaitAll(
		func() { newer.Listen() },
		func() { defer closeResource(newer); <-this.context.Done() },
		func() {
			select {
			case <-this.context.Done():
			case newerIsReady := <-ready:
				if newerIsReady {
					this.log.Printf("[INFO] Pending task [%d] has arrived at a ready state; shutting down previous task, if any.", id)
					closeResource(older)
				} else {
					this.log.Printf("[WARN] Pending task [%d] did not arrive at a ready state; continuing with previous task, if any.", id)
					closeResource(newer)
				}
			}
		},
	)
}

func awaitAll(actions ...func()) (wait func()) {
	var waiter sync.WaitGroup
	for _, action := range actions {
		waiter.Add(1)
		go act(action, waiter.Done)
	}
	return waiter.Wait
}
func act(action, done func()) {
	defer done()
	action()
}

func closeResource(resource io.Closer) {
	if resource != nil {
		_ = resource.Close()
	}
}
