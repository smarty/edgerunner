package edgerunner

import (
	"io"
	"sync"
	"sync/atomic"
	"syscall"
)

type defaultRunner struct {
	configuration
	ready   chan bool
	waiters chan func()
	counter *atomic.Int32
	task    Task
}

func newRunner(configuration configuration) Runner {
	return &defaultRunner{
		configuration: configuration,
		ready:         make(chan bool, 16),
		waiters:       make(chan func()),
		counter:       new(atomic.Int32),
	}
}

func (this *defaultRunner) Close() error {
	this.log.Printf("[INFO] Request to close runner received, shutting down runner along with any associated task(s)...")
	this.cancel()
	return nil
}
func (this *defaultRunner) Reload() {
	this.reloads <- syscall.Signal(0)
}
func (this *defaultRunner) Listen() {
	this.log.Printf("[INFO] Running configured task [%s] at version [%s]...", this.name, this.version)

	go this.listenAll()

	for wait := range this.waiters {
		wait()
	}

	this.log.Printf("[INFO] The configured runner has completed execution of all specified tasks.")
}
func (this *defaultRunner) listenAll() {
	defer close(this.waiters)
	for {
		this.counter.Add(1)
		id := int(this.counter.Load())

		if next := this.prepareNextTask(id); next != nil {
			this.waiters <- this.start(id, next, this.task)
		}

		select {
		case <-this.reloads:
			continue
		case <-this.terminations:
			return
		case <-this.context.Done():
			return
		}
	}
}
func (this *defaultRunner) prepareNextTask(id int) Task {
	next := this.factory(id, this.ready)
	if next == nil {
		this.log.Printf("[WARN] No task created for ID [%d].", id)
		return nil
	}

	err := next.Initialize(this.context)
	if err != nil {
		this.log.Printf("[WARN] Unable to initialize task [%d]: %s", id, err)
		this.closeResource(next)
		return nil
	}

	return next
}
func (this *defaultRunner) start(id int, newer, older Task) (wait func()) {
	this.task = newer
	return awaitAll(
		func() { newer.Listen() },
		func() { defer this.closeResource(newer); <-this.context.Done() },
		func() {
			select {
			case <-this.context.Done():
			case newerIsReady := <-this.ready: // TODO: drain ready.. (maybe make per-task readiness channels)
				if newerIsReady {
					this.log.Printf("[INFO] Pending task [%d] has arrived at a ready state; shutting down previous task, if any.", id)
					this.closeResource(older)
				} else {
					this.log.Printf("[WARN] Pending task [%d] did not arrive at a ready state; continuing with previous task, if any.", id)
					this.closeResource(newer)
				}
			}
		},
	)
}
func awaitAll(actions ...func()) (wait func()) {
	var waiter sync.WaitGroup
	waiter.Add(len(actions))
	for _, action := range actions {
		go func(act func()) {
			defer waiter.Done()
			act()
		}(action)
	}
	return waiter.Wait
}
func (this *defaultRunner) closeResource(resource io.Closer) {
	//this.Logger.Printf("closing resource: %v", resource)
	//dump := debug.Stack()
	//this.Logger.Printf("stack:\n%s", string(dump))
	if resource != nil {
		_ = resource.Close()
	}
}
