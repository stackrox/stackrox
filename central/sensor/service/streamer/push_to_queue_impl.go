package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

type pushToQueueImpl struct {
	onPush   func()
	onFinish func()
}

// Start starts pulling from the input channel and pushing to the queue.
func (s *pushToQueueImpl) Start(inputChannel <-chan *central.MsgFromSensor, toPush pushable) {
	go s.pushLoop(inputChannel, toPush)
}

// pushLoop loops over the input and adds it to the DB or outgoing channel if the DB can be skipped.
func (s *pushToQueueImpl) pushLoop(inputChannel <-chan *central.MsgFromSensor, toPush pushable) {
	defer s.onFinish()

	for {
		if s.pushNext(inputChannel, toPush) {
			s.onPush()
		} else {
			return
		}
	}
}

func (s *pushToQueueImpl) pushNext(inputChannel <-chan *central.MsgFromSensor, toPush pushable) (keepPushing bool) {
	// Try to read the channel. If it is closed, then return false so the loop knows to stop.
	in, ok := <-inputChannel
	if !ok {
		return false
	}

	// If we got something try to push it into the queue, logging any error, and return true since we can keep going.
	if err := toPush.Push(in); err != nil {
		log.Errorf("error pushing event to queue: %s", err)
	}
	return true
}
