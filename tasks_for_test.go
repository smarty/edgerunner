package edgerunner

import (
	"context"
	"sync/atomic"
	"time"
)

type TaskForTests struct {
	log Logger

	id        int
	readiness *bool
	initErr   error

	initialized *atomic.Int32
	listened    *atomic.Int32
	closed      *atomic.Int32
	counter     *atomic.Int32
}

func NewTaskForTests(logger Logger, readiness *bool) *TaskForTests {
	return &TaskForTests{
		log:         logger,
		readiness:   readiness,
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
	for this.closed.Load() <= this.listened.Load() {
		this.log.Printf("listen: %d", this.counter.Add(1))
		time.Sleep(delay() / 5)
	}
}
func (this *TaskForTests) Close() error {
	defer this.closed.Add(1)
	this.log.Printf("closing")
	return nil
}

var (
	prepared      = func() *bool { a := true; return &a }
	unprepared    = func() *bool { a := false; return &a }
	omitReadiness = func() *bool { return nil }
)
