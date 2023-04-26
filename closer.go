package edgerunner

import (
	"io"
	"sync"
)

type closedOnce struct {
	closer func()
	once   *sync.Once
}

func newClosedOnce(closer io.Closer) *closedOnce {
	return &closedOnce{
		closer: func() { closeResource(closer) },
		once:   new(sync.Once),
	}
}
func (this *closedOnce) Close() error {
	this.once.Do(this.closer)
	return nil
}
func closeResource(resource io.Closer) {
	if resource != nil {
		_ = resource.Close()
	}
}
