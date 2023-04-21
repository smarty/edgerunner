package edgerunner

import (
	"context"
	"sync/atomic"
	"time"
)

type TestingTask struct {
	Logger
	initErr  error
	closeErr error

	initialized *atomic.Int32
	listened    *atomic.Int32
	closed      *atomic.Int32
	counter     *atomic.Int32
}

func NewTestingTask(logger Logger) *TestingTask {
	return &TestingTask{
		Logger:      logger,
		initialized: new(atomic.Int32),
		listened:    new(atomic.Int32),
		closed:      new(atomic.Int32),
		counter:     new(atomic.Int32),
	}
}
func (this *TestingTask) Initialize(ctx context.Context) error {
	defer this.initialized.Add(1)
	this.Printf("initializing: %v", ctx.Value("name"))
	return this.initErr
}
func (this *TestingTask) Listen() {
	defer this.listened.Add(1)
	this.Printf("listening")
	for {
		if this.closed.Load() > this.listened.Load() {
			break
		}
		counter := this.counter.Load()
		if counter == -1 {
			break
		}
		this.Printf("listen: %d", counter)
		time.Sleep(time.Millisecond * 100)
		this.counter.Add(1)
	}
}
func (this *TestingTask) Close() error {
	defer this.closed.Add(1)
	this.Printf("closing")
	return this.closeErr
}
