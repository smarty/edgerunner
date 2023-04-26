package edgerunner

import (
	"io"
	"os/signal"
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
	this.id++
	id := this.id
	ready := make(chan bool, 16)
	task := this.taskFactory(this.id, ready)
	if task == nil {
		this.log.Printf("[WARN] No task created for ID [%d].", id)
		return nil
	}

	err := task.Initialize(this.context)
	if err != nil {
		this.log.Printf("[WARN] Unable to initialize task [%d]: %s", id, err)
		closeResource(task)
		return nil
	}

	older := this.task
	newer := newClosedOnce(task)
	this.task = newer

	return prepareWaiter(load(
		func() { task.Listen() },
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
