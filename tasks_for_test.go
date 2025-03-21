package edgerunner

import (
	"context"
	"sync/atomic"
	"time"
)

type TaskForTests struct {
	log logger

	id        int
	readiness *bool
	initErr   error

	initialized *atomic.Int32
	listened    *atomic.Int32
	closed      *atomic.Int32
	counter     *atomic.Int32
	ctx         context.Context
	ctxError    error
}

func NewTaskForTests(logger logger, readiness *bool) *TaskForTests {
	return &TaskForTests{
		log:         logger,
		readiness:   readiness,
		initialized: new(atomic.Int32),
		listened:    new(atomic.Int32),
		closed:      new(atomic.Int32),
		counter:     new(atomic.Int32),
	}
}
func (this *TaskForTests) Initialize(ctx context.Context) error {
	defer this.initialized.Add(1)
	this.log.Printf("initializing")
	this.ctx = ctx
	return this.initErr
}
func (this *TaskForTests) identify(id int, ready func(bool)) {
	this.id = id
	if this.readiness != nil {
		ready(*this.readiness)
		ready(false) // ensure runner ignores (and doesn't choke on) repeated calls
		ready(true)  // ensure runner ignores (and doesn't choke on) repeated calls
	}
}
func (this *TaskForTests) Listen() {
	defer this.log.Printf("done listening")
	defer this.listened.Add(1)
	this.log.Printf("listening")
	for this.closed.Load() <= this.listened.Load() {
		this.log.Printf("listen: %d", this.counter.Add(1))
		time.Sleep(delay() / 5)
	}
}
func (this *TaskForTests) Close() error {
	defer this.log.Printf("done closing")
	defer this.closed.Add(1)
	this.log.Printf("will close after shutdown delay")
	child, cancel := context.WithTimeout(this.ctx, time.Millisecond*20)
	defer cancel()
	<-child.Done()
	this.ctxError = child.Err()
	return nil
}

var (
	prepared      = func() *bool { a := true; return &a }
	unprepared    = func() *bool { a := false; return &a }
	omitReadiness = func() *bool { return nil }
)
