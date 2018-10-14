package streamer

import (
	"sync"

	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/queue"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
)

type streamerImpl struct {
	finishedSending concurrency.Signal

	clusterID string
	qu        queue.EventQueue
	pl        pipeline.Pipeline

	lock               sync.RWMutex
	eventsRead         chan *v1.SensorEvent
	eventsQueued       chan *v1.SensorEvent
	enforcementsToSend chan *v1.SensorEnforcement
}

// Start sets up the channels and signals to start processing events input through the given stream, and return
// enforcement actions to the given stream.
func (s *streamerImpl) Start(stream v1.SensorEventService_RecordEventServer) {
	s.readFromStream(stream)
	s.enqueueDequeue()
	s.processWithPipeline()
	s.sendToStream(stream)
}

// WaitUntilEmpty waits until all items input from the sensor stream have been processed.
func (s *streamerImpl) WaitUntilFinished() {
	s.finishedSending.Wait()
}

// InjectEvent tries to add the event to the stream that is processed and generates enforcements and returns whether or
// not it was successful.
func (s *streamerImpl) InjectEvent(event *v1.SensorEvent) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.eventsRead == nil {
		return false
	}
	s.eventsRead <- event
	return true
}

// InjectEnforcement tries to add the enforcement to the stream sent to sensor and returns whether or not it was
// successful.
func (s *streamerImpl) InjectEnforcement(enforcement *v1.SensorEnforcement) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.enforcementsToSend == nil {
		return false
	}
	s.enforcementsToSend <- enforcement
	return true
}

// readFromStream reads from the given stream and forwards data to the output event channel.
// When the stream is closed or the context canceled, the channel is closed.
func (s *streamerImpl) readFromStream(stream v1.SensorEventService_RecordEventServer) {
	s.eventsRead = make(chan *v1.SensorEvent)
	whenReadingFinishes := func() {
		s.lock.Lock()
		defer s.lock.Unlock()

		close(s.eventsRead)
		s.eventsRead = nil
	}

	// Sensor -> InputChannel
	NewReceiver(s.clusterID, whenReadingFinishes).Start(stream, s.eventsRead)
}

// enqueueDequeue reads from the given channel, and queues the inputs, outputing them on the returned channel.
// The output channel is closed when the input channel is closed, and the queue is empty.
func (s *streamerImpl) enqueueDequeue() {
	s.eventsQueued = make(chan *v1.SensorEvent)
	closeOutput := func() {
		close(s.eventsQueued)
	}

	turnstile := concurrency.NewTurnstile()
	closeTurnstile := func() { turnstile.Close() }

	// InputChannel -> Queue
	// Whenever a new item is added to the Queue call AllowOne, to make sure the turnstile is primed to allow the puller
	// dequeue again.
	NewPushToQueue(turnstile.AllowOne, closeTurnstile).Start(s.eventsRead, s.qu)

	// Queue -> Processing
	// In between emptying the queue, wait for the pusher to signal (call Wait) and only continue if true is returned.
	NewPullFromQueue(turnstile.Wait, closeOutput).Start(s.qu, s.eventsQueued)
}

// processWithPipeline reads data from the input channel, processes it with the pipeline configured for the streamer,
// and outputs any results (errors are logged internally) to the output channel.
// The output channel is closed when the input channel is closed.
func (s *streamerImpl) processWithPipeline() {
	s.enforcementsToSend = make(chan *v1.SensorEnforcement)
	closeOutput := func() {
		s.lock.Lock()
		defer s.lock.Unlock()

		close(s.enforcementsToSend)
		s.enforcementsToSend = nil
	}

	// Processing -> Process -> OutputChannel
	NewPipeline(closeOutput).Start(s.eventsQueued, s.pl, s)
}

// sendToStream reads from the input channel and sends received data out over the input stream. When the input channel
// is closed, and all data is sent out over the stream, the output signal is signalled.
func (s *streamerImpl) sendToStream(stream v1.SensorEventService_RecordEventServer) {
	s.finishedSending = concurrency.NewSignal()
	sendFinishedSignal := func() { s.finishedSending.Signal() }

	// OutputChannel -> Sensor
	NewSender(sendFinishedSignal).Start(s.enforcementsToSend, stream)
}
