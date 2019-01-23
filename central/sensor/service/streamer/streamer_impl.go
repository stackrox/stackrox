package streamer

import (
	"sync"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/queue"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

type streamerImpl struct {
	finishedSending concurrency.Signal

	clusterID string
	qu        queue.Queue
	pl        pipeline.Pipeline

	lock      sync.RWMutex
	msgsRead  chan *central.MsgFromSensor
	msgQueued chan *central.MsgFromSensor
	msgToSend chan *central.MsgToSensor

	stopSig    concurrency.ErrorSignal
	stoppedSig concurrency.ErrorSignal
}

// Start sets up the channels and signals to start processing events input through the given stream, and return
// enforcement actions to the given stream.
func (s *streamerImpl) Start(server central.SensorService_CommunicateServer) {
	s.readFromStream(server)
	s.enqueueDequeue()
	s.processWithPipeline()
	s.sendToStream(server)
}

// WaitUntilEmpty waits until all items input from the sensor stream have been processed.
func (s *streamerImpl) WaitUntilFinished() error {
	return s.stoppedSig.Wait()
}

// InjectEnforcement tries to add the enforcement to the stream sent to sensor and returns whether or not it was
// successful.
func (s *streamerImpl) InjectMessage(msg *central.MsgToSensor) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.msgToSend == nil {
		return false
	}
	s.msgToSend <- msg
	return true
}

// readFromStream reads from the given stream and forwards data to the output event channel.
// When the stream is closed or the context canceled, the channel is closed.
func (s *streamerImpl) readFromStream(server central.SensorService_CommunicateServer) {
	s.msgsRead = make(chan *central.MsgFromSensor)
	whenReadingFinishes := func() {
		s.lock.Lock()
		defer s.lock.Unlock()

		close(s.msgsRead)
		s.msgsRead = nil
	}

	// Sensor -> InputChannel
	NewReceiver(s.clusterID, whenReadingFinishes).Start(server, s.msgsRead)
}

// enqueueDequeue reads from the given channel, and queues the inputs, outputing them on the returned channel.
// The output channel is closed when the input channel is closed, and the queue is empty.
func (s *streamerImpl) enqueueDequeue() {
	s.msgQueued = make(chan *central.MsgFromSensor)
	closeOutput := func() {
		close(s.msgQueued)
	}

	turnstile := concurrency.NewTurnstile()
	closeTurnstile := func() { turnstile.Close() }

	// InputChannel -> Queue
	// Whenever a new item is added to the Queue call AllowOne, to make sure the turnstile is primed to allow the puller
	// dequeue again.
	NewPushToQueue(turnstile.AllowOne, closeTurnstile).Start(s.msgsRead, s.qu)

	// Queue -> Processing
	// In between emptying the queue, wait for the pusher to signal (call Wait) and only continue if true is returned.
	NewPullFromQueue(turnstile.Wait, closeOutput).Start(s.qu, s.msgQueued)
}

// processWithPipeline reads data from the input channel, processes it with the pipeline configured for the streamer,
// and outputs any results (errors are logged internally) to the output channel.
// The output channel is closed when the input channel is closed.
func (s *streamerImpl) processWithPipeline() {
	s.msgToSend = make(chan *central.MsgToSensor)
	closeOutput := func(err error) {
		s.lock.Lock()
		defer s.lock.Unlock()

		close(s.msgToSend)
		s.msgToSend = nil

		s.stoppedSig.SignalWithError(err)
	}

	// Processing -> Process -> OutputChannel
	NewPipeline(closeOutput).Start(s.msgQueued, s.pl, s, &s.stopSig)
}

// sendToStream reads from the input channel and sends received data out over the input stream. When the input channel
// is closed, and all data is sent out over the stream, the output signal is signalled.
func (s *streamerImpl) sendToStream(server central.SensorService_CommunicateServer) {
	s.finishedSending = concurrency.NewSignal()
	sendFinishedSignal := func() { s.finishedSending.Signal() }

	// OutputChannel -> Sensor
	NewSender(sendFinishedSignal).Start(s.msgToSend, server)
}

func (s *streamerImpl) Terminate(err error) bool {
	stopped := s.stopSig.SignalWithError(err)
	s.stoppedSig.Wait()
	return stopped
}
