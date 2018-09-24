package streamer

import (
	"github.com/stackrox/rox/generated/api/v1"
)

type pullFromQueueImpl struct {
	onEmpty  func() bool
	onFinish func()
}

// Start starts pulling from the queue and pushing to the output channel.
func (s *pullFromQueueImpl) Start(toPull pullable, outputChannel chan<- *v1.SensorEvent) {
	go s.pullLoop(toPull, outputChannel)
}

// pullLoop grabs the next available output and pushes it to the channel when able.
func (s *pullFromQueueImpl) pullLoop(toPull pullable, outputChannel chan<- *v1.SensorEvent) {
	// notification that the loop has been exited.
	defer s.onFinish()

	// onEmpty returns if we should try pulling again when we've emptied the queue.
	for s.onEmpty() {
		s.pullUntilEmpty(toPull, outputChannel)
	}
}

func (s *pullFromQueueImpl) pullUntilEmpty(toPull pullable, outputChannel chan<- *v1.SensorEvent) {
	for {
		// Pull the next item from the queue.
		next, err := toPull.Pull()

		// If there was an error pulling, just go back and try again.
		if err != nil {
			log.Errorf("error pulling event from queue: %s", err)
			continue
		}

		// If we pulled and got nil with no error, the queue is currently empty, so go back to wait.
		if next == nil {
			return
		}

		// We got something from the queue, push it to the channel.
		outputChannel <- next
	}
}
