package streamer

import (
	"errors"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

type enqueueDequeueImpl struct {
	queue *queueImpl

	output chan *central.MsgFromSensor

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

// Start starts pulling from the input channel and pushing to the queue.
func (s *enqueueDequeueImpl) Start(input <-chan *central.MsgFromSensor, dependents ...Stoppable) {
	go s.run(input, dependents...)
}

func (s *enqueueDequeueImpl) Stop(err error) bool {
	return s.stopC.SignalWithError(err)
}

func (s *enqueueDequeueImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

func (s *enqueueDequeueImpl) Output() <-chan *central.MsgFromSensor {
	return s.output
}

func (s *enqueueDequeueImpl) run(input <-chan *central.MsgFromSensor, dependents ...Stoppable) {
	defer func() {
		s.stoppedC.SignalWithError(s.stopC.Err())
		StopAll(s.stoppedC.Err(), dependents...)
	}()

	go s.enqueue(input)
	s.dequeue()
}

func (s *enqueueDequeueImpl) enqueue(input <-chan *central.MsgFromSensor, dependents ...Stoppable) {
	for !s.stopC.IsDone() {
		select {
		case in, ok := <-input:
			if !ok {
				s.stopC.SignalWithError(errors.New("channel unexpectedly closed"))
				return
			}
			s.queue.push(in)

		case <-s.stopC.Done():
			return
		}
	}
}

func (s *enqueueDequeueImpl) dequeue() {
	for !s.stopC.IsDone() {
		select {
		case <-s.queue.notEmpty():
			msg := s.queue.pull()
			if msg == nil {
				errorhelpers.PanicOnDevelopment(errors.New("dequeued when queue was empty"))
			}
			if !s.writeToOutput(msg) {
				log.Debugf("message received from queue dropped: %+v", msg)
			}

		case <-s.stopC.Done():
			return
		}
	}
}

func (s *enqueueDequeueImpl) writeToOutput(out *central.MsgFromSensor) bool {
	select {
	case s.output <- out:
		return true
	case <-s.stoppedC.Done():
		return false
	}
}
