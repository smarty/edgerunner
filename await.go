package edgerunner

import "sync"

func load(items ...func()) chan func() {
	output := make(chan func())
	go func() {
		defer close(output)
		for _, item := range items {
			output <- item
		}
	}()
	return output
}
func awaitAll(actions chan func()) {
	waiter := prepareWaiter(actions)
	waiter()
}
func prepareWaiter(actions chan func()) (wait func()) {
	var waiter sync.WaitGroup
	for action := range actions {
		if action != nil {
			waiter.Add(1)
			go do(action, waiter.Done)
		}
	}
	return waiter.Wait
}
func do(action, done func()) {
	defer done()
	action()
}
