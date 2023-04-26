package edgerunner

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

type TaskForTests struct {
	log Logger

	readiness *bool
	initErr   error
	closeErr  error

	initialized *atomic.Int32
	listened    *atomic.Int32
	closed      *atomic.Int32
	counter     *atomic.Int32

	id int
}

func NewTaskForTests(logger Logger) *TaskForTests {
	return &TaskForTests{
		log:         logger,
		initialized: new(atomic.Int32),
		listened:    new(atomic.Int32),
		closed:      new(atomic.Int32),
		counter:     new(atomic.Int32),
	}
}
func (this *TaskForTests) Initialize(ctx context.Context) error {
	defer this.initialized.Add(1)
	this.log.Printf("initializing: %v", ctx.Value("name"))
	return this.initErr
}
func (this *TaskForTests) identify(id int, ready chan<- bool) {
	this.id = id
	if this.readiness != nil {
		ready <- *this.readiness
	}
}
func (this *TaskForTests) Listen() {
	defer this.log.Printf("done listening")
	defer this.listened.Add(1)
	this.log.Printf("listening")
	for {
		if this.closed.Load() > this.listened.Load() {
			break
		}
		counter := this.counter.Load()
		if counter == -1 {
			break
		}
		this.log.Printf("listen: %d", counter)
		time.Sleep(time.Millisecond * 100)
		this.counter.Add(1)
	}
}
func (this *TaskForTests) Close() error {
	defer this.closed.Add(1)
	this.log.Printf("closing")
	return this.closeErr
}
func (this *TaskForTests) String() string {
	return fmt.Sprintf("task %d; initialized: %d; counter %d; listened: %d; closed %d",
		this.id, this.initialized.Load(), this.counter.Load(), this.listened.Load(), this.closed.Load())
}

var (
	prepared   = func() *bool { a := true; return &a }
	unprepared = func() *bool { a := false; return &a }
)
