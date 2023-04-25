package edgerunner

import (
	"io"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

type defaultRunner struct {
	configuration
	id   *atomic.Int32
	task Task
}

func newRunner(config configuration) Runner {
	runner := &defaultRunner{id: new(atomic.Int32)}
	runner.configuration = config
	return runner
}

func (this *defaultRunner) Listen() {
	this.log.Printf("[INFO] Running configured task [%s] at version [%s]...", this.name, this.version)
	awaitAll(this.goCoordinateTaskWithSignals())
	this.log.Printf("[INFO] The configured runner has completed execution of all specified tasks.")
}
func (this *defaultRunner) goCoordinateTaskWithSignals() chan func() {
	tasks := make(chan func())
	go func() {
		defer close(tasks)
		for {
			tasks <- this.startNextTask(this.task)

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
	}()
	return tasks
}
func (this *defaultRunner) startNextTask(older Task) (task func()) {
	id := int(this.id.Add(1))
	ready := make(chan bool, 16)
	next := this.factory(id, ready)
	if next == nil {
		this.log.Printf("[WARN] No task created for ID [%d].", id)
		return nil
	}

	err := next.Initialize(this.context)
	if err != nil {
		this.log.Printf("[WARN] Unable to initialize task [%d]: %s", id, err)
		closeResource(next)
		return nil
	}

	this.task = next

	return prepareWaiter(load(
		func() { next.Listen() },
		func() { defer closeResource(next); <-this.context.Done() },
		func() {
			select {
			case <-this.context.Done():
			case newerIsReady := <-ready:
				if newerIsReady {
					this.log.Printf("[INFO] Pending task [%d] has arrived at a ready state; shutting down previous task, if any.", id)
					closeResource(older)
				} else {
					this.log.Printf("[WARN] Pending task [%d] did not arrive at a ready state; continuing with previous task, if any.", id)
					closeResource(next)
				}
			}
		},
	))
}

func (this *defaultRunner) Reload() {
	select {
	case <-this.context.Done(): // already shut down
	case this.reloads <- syscall.Signal(0):
	default: // reloads chan may be full
	}
}

func (this *defaultRunner) Close() error {
	this.log.Printf("[INFO] Request to close runner received, shutting down runner along with any associated task(s)...")
	this.cancel()
	signal.Stop(this.terminations)
	signal.Stop(this.reloads)
	return nil
}
func closeResource(resource io.Closer) {
	if resource != nil {
		_ = resource.Close()
	}
}

func load(items ...func()) chan func() {
	output := make(chan func())
	go func() {
		defer close(output)
		for _, item := range items {
			output <- item
		}
	}()
	return output
}
func awaitAll(actions chan func()) {
	waiter := prepareWaiter(actions)
	waiter()
}
func prepareWaiter(actions chan func()) (wait func()) {
	var waiter sync.WaitGroup
	for action := range actions {
		if action != nil {
			waiter.Add(1)
			go do(action, waiter.Done)
		}
	}
	return waiter.Wait
}
func do(action, done func()) {
	defer done()
	action()
}
