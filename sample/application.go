package main

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/smarty/edgerunner"
)

type Application struct {
	log    edgerunner.Logger
	config ConfigFile
	closed *atomic.Bool
	ready  chan<- bool
	done   <-chan struct{}
}

func NewApplication(logger edgerunner.Logger, ready chan<- bool) *Application {
	return &Application{
		log:    logger,
		ready:  ready,
		closed: new(atomic.Bool),
	}
}

func (this *Application) Initialize(ctx context.Context) (err error) {
	this.done = ctx.Done()
	this.config, err = LoadConfig("config.json")
	return err
}

func (this *Application) Listen() {
	this.ready <- true
	for {
		if this.closed.Load() {
			this.log.Printf("task closed, exiting listen")
			break
		}

		select {
		case <-this.done:
			this.log.Printf("context cancelled, exiting Listen")
			break
		default:
			this.log.Printf(this.config.Message)
		}

		time.Sleep(time.Second)
	}
}
func (this *Application) Close() error {
	this.log.Printf("closing task")
	this.closed.Store(true)
	return nil
}
