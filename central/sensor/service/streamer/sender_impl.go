package streamer

import (
	"errors"
	"io"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Service is the struct that manages the SensorEvent API
type senderImpl struct {
	injected chan *central.MsgToSensor

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

// Start sets up the channels and signal to start processing events input through the given stream, and return
// enforcement actions to the given stream.
func (s *senderImpl) Start(server central.SensorService_CommunicateServer, dependents ...Stoppable) {
	go s.pipelineToSend(server, dependents...)
}

func (s *senderImpl) Stop(err error) bool {
	return s.stopC.SignalWithError(err)
}

func (s *senderImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

func (s *senderImpl) InjectMessage(in *central.MsgToSensor) bool {
	select {
	case s.injected <- in:
		return true
	case <-s.stoppedC.Done():
		return false
	}
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *senderImpl) pipelineToSend(server central.SensorService_CommunicateServer, dependents ...Stoppable) {
	defer func() {
		s.stoppedC.SignalWithError(s.stopC.Err())
		StopAll(s.stoppedC.Err(), dependents...)
	}()

	for !s.stopC.IsDone() {
		select {
		case msg, ok := <-s.injected:
			// Looping stops when the output from pending events closes.
			if !ok {
				s.stopC.SignalWithError(errors.New("input channel unexpectedly closed"))
				return
			}

			log.Debugf("sending message to sensor: %+v", msg)
			err := server.Send(msg)

			if err == io.EOF {
				s.stopC.SignalWithError(errors.New("stream send unexpectedly closed"))
				return
			} else if err != nil {
				log.Error("error sending deployment enforcement", err)
			}

		case <-s.stopC.Done():
			return
		}
	}
}
