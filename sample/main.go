package main

import (
	"fmt"
	"log"

	"github.com/smarty/edgerunner"
)

func main() {
	runner := edgerunner.New(
		edgerunner.Options.TaskName("SAMPLE"),
		edgerunner.Options.TaskVersion("DEV"),
		edgerunner.Options.Logger(log.New(log.Writer(), "EDGE   ", log.Lmicroseconds)),
		edgerunner.Options.TaskFactory(func(id int, ready chan<- bool) edgerunner.Task {
			taskLogger := log.New(log.Writer(), fmt.Sprintf("TASK-%d ", id), log.Lmicroseconds)
			return NewApplication(taskLogger, ready)
		}),
	)
	runner.Listen()
}
