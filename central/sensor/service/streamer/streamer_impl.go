package streamer

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

type streamerImpl struct {
	receiver       Receiver
	enqueueDequeue EnqueueDequeue
	pipeline       Pipeline
	sender         Sender
}

// Start sets up the channels and signals to start processing events input through the given stream, and return
// enforcement actions to the given stream.
func (s *streamerImpl) Start(server central.SensorService_CommunicateServer) {
	// Receiver is only stopped by closing the server.
	s.receiver.Start(server, s.enqueueDequeue)

	// These can all be stopped at will. We pass the stages before and after each other as dependents, so if
	// any stage is closed, all upstream and downstream stages get stopped as well.
	s.enqueueDequeue.Start(s.receiver.Output(), s.pipeline)
	s.pipeline.Start(s.enqueueDequeue.Output(), s.sender, s.enqueueDequeue, s.sender)
	s.sender.Start(server, s.pipeline)
}

func (s *streamerImpl) Terminate(err error) bool {
	// We consider stopping the pipeline termination since nothing will get processed/updated after that stops.
	stopped := s.pipeline.Stop(err)
	_ = s.pipeline.Stopped().Wait()
	return stopped
}

// WaitUntilEmpty waits until all items input from the sensor stream have been processed, and any resulting
// responses have been sent back.
func (s *streamerImpl) WaitUntilFinished() error {
	return s.sender.Stopped().Wait()
}

// InjectEnforcement tries to add the enforcement to the stream sent to sensor and returns whether or not it was
// successful.
func (s *streamerImpl) InjectMessage(msg *central.MsgToSensor) bool {
	return s.sender.InjectMessage(msg)
}
