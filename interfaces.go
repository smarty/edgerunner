package edgerunner

import (
	"context"
	"io"
)

type (
	ListenCloser interface {
		Listen()
		io.Closer
	}
	Runner interface {
		ListenCloser
		Reload()
	}
)

type (
	Task interface {
		Initialize(ctx context.Context) error
		ListenCloser
	}
	TaskFactory func(id int, ready chan<- bool) Task
)

type Monitor interface {
	// TODO
}
type Logger interface {
	Printf(string, ...interface{})
}
