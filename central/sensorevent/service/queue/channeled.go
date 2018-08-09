package queue

import (
	"github.com/stackrox/rox/central/sensorevent/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// ChanneledEventQueue provides a dual channel interface to a queue of elements that need to be processed.
// All elements buffered in the queue (not available on the outgoing channel), get stored in a provided
// storage instance, and added to the outgoing channel as it is consumed.
type ChanneledEventQueue interface {
	InChannel() chan<- *v1.SensorEvent
	OutChannel() <-chan *v1.SensorEvent
	Open(clusterID string) error
	Close()
}

// NewChanneledEventQueue returns a new instance of the ChanneledEventQueue interface, specifically, one that persists
// all events currently in the queue.
func NewChanneledEventQueue(eventStorage store.Store) ChanneledEventQueue {
	return &channeledPersistedEventQueue{
		queue: NewPersistedEventQueue(eventStorage),

		stopLoop:    make(chan struct{}),
		loopStopped: make(chan struct{}),
	}
}

// channeledPersistedEventQueue is an implementation of the ChanneledEventQueue interface that is implemented by wrapping
// a persistentEventQueue with channel push and pull operations.
type channeledPersistedEventQueue struct {
	queue EventQueue

	inputChannel  chan *v1.SensorEvent
	outputChannel chan *v1.SensorEvent

	pendingChannel chan struct{}
	stopLoop       chan struct{}
	loopStopped    chan struct{}
}

// InChannel returns the write-only channel that adds items to the queue.
func (s *channeledPersistedEventQueue) InChannel() chan<- *v1.SensorEvent {
	return s.inputChannel
}

// OutChannel returns the read-only channel that pulls items from the queue.
func (s *channeledPersistedEventQueue) OutChannel() <-chan *v1.SensorEvent {
	return s.outputChannel
}

// Open starts the reading and writing to the in and out channels, first loading all elements in the DB for the given cluster.
func (s *channeledPersistedEventQueue) Open(clusterID string) error {
	if err := s.queue.Load(clusterID); err != nil {
		return err
	}

	s.inputChannel = make(chan *v1.SensorEvent)
	s.pendingChannel = make(chan struct{}, 1)
	s.outputChannel = make(chan *v1.SensorEvent)

	go s.pushLoop()
	go s.pullLoop()
	return nil
}

// Close stops the reading and writing from in and out channels.
func (s *channeledPersistedEventQueue) Close() {
	close(s.inputChannel)
	<-s.loopStopped
	close(s.pendingChannel)
	<-s.loopStopped
	close(s.outputChannel)
}

// pushLoop loops over the input and adds it to the DB or outgoing channel if the DB can be skipped.
func (s *channeledPersistedEventQueue) pushLoop() {
	// notification that the loop has been exited.
	defer func() { s.loopStopped <- struct{}{} }()

	for {
		// Looping stops when we close the input channel.
		in, ok := <-s.inputChannel
		if !ok {
			return
		}

		if err := s.queue.Push(in); err != nil {
			log.Errorf("unable to push to queue: %s", err)
			continue
		}

		s.thereMightBeMoreQueued()
	}
}

// pullLoop grabs the next available output and pushes it to the channel when able.
func (s *channeledPersistedEventQueue) pullLoop() {
	// notification that the loop has been exited.
	defer func() { s.loopStopped <- struct{}{} }()

	for {
		// Looping stops when the pending channel closes.
		_, ok := <-s.pendingChannel
		if !ok {
			return
		}

		if s.queue.Count() == 0 {
			continue
		}

		next, err := s.queue.Pull()
		if err != nil {
			log.Errorf("unable to pull from queue: %s", err)
			continue
		}
		if next == nil {
			continue
		}

		s.outputChannel <- next
		s.thereMightBeMoreQueued()
	}
}

// Make pending kicks off an output cycle if one is not already in action.
func (s *channeledPersistedEventQueue) thereMightBeMoreQueued() {
	select {
	case s.pendingChannel <- struct{}{}:
		return
	default:
		return
	}
}
