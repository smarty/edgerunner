package edgerunner

import (
	"context"
	"io"
)

type (
	ListenCloser interface {
		// Listen starts a long-lived component.
		// It is the caller's responsibility to call Listen on the Runner.
		// It is the Runner's responsibility to call Listen on Task values
		// provided via the TaskFactory.
		Listen()

		// Closer should cause the Listen method to terminate.
		// The Runner waits until Listen has finished.
		io.Closer
	}

	Runner interface {
		ListenCloser

		// Reload sends a signal to the Runner to invoke the supplied
		// TaskFactory to build a new Task instance. If the new Task
		// initializes without error and indicates its readiness via
		// the supplied `chan<- bool` (see the TaskFactory) the runner
		// will call Close on the previously running Task (if any).
		// Both the old and new Task values will be running simultaneously
		// for a short time (by design).
		Reload()
	}
)

type (
	// Task describes a type supplied by the caller, via the TaskFactory.
	// Generally a task is a long-lived component (like an HTTP server).
	Task interface {
		// Initialize provides an opportunity for the task to load its
		// configuration, incorporating the supplied context.Context for
		// operations that support it.
		Initialize(ctx context.Context) error
		ListenCloser
	}
	// TaskFactory is a callback for building whatever task will be managed
	// by the Runner. The supplied task should call the ready func to indicate
	// its readiness to begin working. Only the first call has any effect.
	TaskFactory func(id int, ready func(bool)) Task
)

type Logger interface {
	Printf(string, ...any)
}
