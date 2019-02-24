package sensor

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"google.golang.org/grpc"
)

// sensor implements the Sensor interface by sending inputs to central,
// and providing the output from central asynchronously.
type centralCommunicationImpl struct {
	receiver CentralReceiver
	sender   CentralSender

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func (s *centralCommunicationImpl) Start(conn *grpc.ClientConn) {
	go s.sendEvents(central.NewSensorServiceClient(conn), s.receiver.Stop, s.sender.Stop)
}

func (s *centralCommunicationImpl) Stop(err error) {
	s.stopC.SignalWithError(err)
}

func (s *centralCommunicationImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

func (s *centralCommunicationImpl) sendEvents(client central.SensorServiceClient, onStops ...func(error)) {
	defer func() {
		s.stoppedC.SignalWithError(s.stopC.Err())
		runAll(s.stopC.Err(), onStops...)
	}()

	// Start the stream client.
	///////////////////////////
	stream, err := client.Communicate(context.Background())
	if err != nil {
		s.stopC.SignalWithError(fmt.Errorf("opening stream: %v", err))
		return
	}
	defer func() {
		if err := stream.CloseSend(); err != nil {
			log.Errorf("Failed to close stream cleanly: %v", err)
		}
	}()
	log.Infof("Established connection to Central.")

	// Start receiving and sending with central.
	////////////////////////////////////////////
	s.receiver.Start(stream, s.Stop, s.sender.Stop)
	s.sender.Start(stream, s.Stop, s.receiver.Stop)
	log.Infof("Communication with central started.")

	// Wait for stop.
	/////////////////
	_ = s.stopC.Wait()
	log.Infof("Communication with central ended.")
}

func runAll(err error, fs ...func(error)) {
	for _, f := range fs {
		f(err)
	}
}
