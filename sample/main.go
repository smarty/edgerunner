package main

import (
	"context"
	"log"
	"syscall"

	"github.com/smarty/edgerunner"
)

func main() {
	runner := edgerunner.New(
		edgerunner.Options.TaskName("task"),
		edgerunner.Options.Logger(log.Default()),
		edgerunner.Options.WatchTerminateSignals(syscall.SIGINT),
		edgerunner.Options.WatchReloadSignals(syscall.SIGHUP),
		edgerunner.Options.Context(context.Background()),
		edgerunner.Options.TaskFactory(newTask),
	)

	runner.Listen()
}

type BasicTask struct {
	id    int
	ready chan<- bool
}

func newTask(id int, ready chan<- bool) edgerunner.Task {
	log.Println("New task", id)
	return &BasicTask{id: id, ready: ready}
}

func (this *BasicTask) Initialize(_ context.Context) error {
	log.Println("Initializing", this.id)
	return nil
}
func (this *BasicTask) Listen() {
	log.Println("Listening", this.id)
	this.ready <- true
}
func (this *BasicTask) Close() error {
	log.Println("Closing", this.id)
	return nil
}
