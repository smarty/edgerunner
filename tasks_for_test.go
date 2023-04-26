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
func (this *TaskForTests) Initialize(_ context.Context) error {
	defer this.initialized.Add(1)
	this.log.Printf("initializing")
	return this.initErr
}
func (this *TaskForTests) identify(id int, ready func(bool)) {
	this.id = id
	if this.readiness != nil {
		ready(*this.readiness)
		ready(false) // ensure runner ignores (and doesn't choke on) repeated calls
		ready(true)  // ensure runner doesn't (and doesn't choke on) repeated calls
	}
}
func (this *TaskForTests) Listen() {
	defer this.log.Printf("done listening")
	defer this.listened.Add(1)
	this.log.Printf("listening")
	for this.closed.Load() <= this.listened.Load() {
		this.log.Printf("listen: %d", this.counter.Load())
		time.Sleep(delay() / 5)
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
