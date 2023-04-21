package edgerunner

import (
	"context"
	"io"
)

type defaultRunner struct {
	ctx         context.Context
	cancel      context.CancelFunc
	taskName    string
	taskVersion string
	taskFactory TaskFactory
	identifier  int
	logger      Logger
}

func newRunner(config configuration) Runner {
	ctx, cancel := context.WithCancel(config.Context)
	return &defaultRunner{
		ctx:         ctx,
		cancel:      cancel,
		taskName:    config.TaskName,
		taskVersion: config.TaskVersion,
		taskFactory: config.TaskFactory,
		logger:      config.Logger,
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
	this.logger.Printf("[INFO] Running configured task [%s] at version [%s]...", this.taskName, this.taskVersion)
	defer this.logger.Printf("[INFO] The configured runner has completed execution of all specified tasks.")

	this.identifier++

	task := this.taskFactory(this.identifier, nil) // TODO: pass ready channel
	if task == nil {
		this.logger.Printf("[WARN] No task created for ID [%d].", this.identifier)
		return
	}

	err := task.Initialize(this.ctx)
	if err != nil {
		this.logger.Printf("[WARN] Unable to initialize task [%d]: %s", this.identifier, err)
		closeResource(task)
		return
	}

	go func() {
		<-this.ctx.Done()
		closeResource(task)
	}()

	task.Listen()
}

func closeResource(resource io.Closer) {
	if resource != nil {
		_ = resource.Close()
	}
}
