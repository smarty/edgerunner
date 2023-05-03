package edgerunner

import (
	"context"
	"io"
	"os/signal"
	"sync"
	"syscall"
)

type defaultRunner struct {
	configuration
	id   int
	task io.Closer
}

func newRunner(config configuration) Runner {
	return &defaultRunner{configuration: config}
}

func (this *defaultRunner) Listen() {
	this.log.Printf("[INFO] Running configured task [%s] at version [%s]...", this.taskName, this.taskVersion)
	tasks := make(chan func())
	go this.coordinateTasksWithSignals(tasks)
	awaitAll(tasks)
	this.log.Printf("[INFO] The configured runner has completed execution of all specified tasks.")
}
func (this *defaultRunner) coordinateTasksWithSignals(tasks chan func()) {
	defer close(tasks)
	for {
		tasks <- this.startNextTask()

		select {
		case value := <-this.reloadSignals:
			this.log.Printf("[INFO] Received OS reload signal [%v], instructing runner to reload configured task...", value)
			continue
		case value := <-this.terminationSignals:
			this.log.Printf("[INFO] Received OS terminate signal [%v], shutting down runner along with any associated task(s)...", value)
			this.cancel()
			return
		case <-this.context.Done():
			return
		}
	}
}
func (this *defaultRunner) startNextTask() (taskWaiter func()) {
	id := this.id // prevent data races by NOT passing this.id to goroutine functions below
	readiness := make(chan bool, 1)
	var once sync.Once
	ready := func(state bool) { once.Do(func() { defer close(readiness); readiness <- state }) }
	task := this.taskFactory(id, ready)
	if task == nil {
		this.log.Printf("[WARN] No task created for ID [%d].", id)
		return nil
	}

	ctx, release := context.WithTimeout(this.context, this.readinessTimeout)
	err := task.Initialize(this.context)
	if err != nil {
		this.log.Printf("[WARN] Unable to initialize task [%d]: %s", id, err)
		release()
		closeResource(task)
		return nil
	}

	older := this.task
	newer := newClosedOnce(task) // prevent data races by NOT passing this.task to goroutine functions below
	this.id++
	this.task = newer

	return prepareWaiter(load(
		func() { defer release(); task.Listen(); this.log.Printf("[INFO] Task [%d] is finished.", id) },
		func() { defer closeResource(newer); <-this.context.Done() },
		func() {
			select {
			case <-ctx.Done():
				this.log.Printf("[INFO] Pending task [%d] failed to report readiness before configured timeout of [%s]; continuing with previous task, if any.", id, this.readinessTimeout)
				closeResource(newer)
			case newerIsReady := <-readiness:
				if newerIsReady {
					this.log.Printf("[INFO] Pending task [%d] has arrived at a ready state; shutting down previous task, if any.", id)
					closeResource(older)
				} else {
					this.log.Printf("[WARN] Pending task [%d] did not arrive at a ready state; continuing with previous task, if any.", id)
					closeResource(newer)
				}
			}
		},
	))
}

func (this *defaultRunner) Reload() {
	select {
	case <-this.context.Done(): // already shut down
	case this.reloadSignals <- syscall.Signal(0):
	default: // reloads chan may be full
	}
}

func (this *defaultRunner) Close() error {
	this.log.Printf("[INFO] Request to close runner received, shutting down runner along with any associated task(s)...")
	this.cancel()
	signal.Stop(this.terminationSignals)
	signal.Stop(this.reloadSignals)
	return nil
}
