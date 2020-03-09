package sensor

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"google.golang.org/grpc"
)

// sensor implements the Sensor interface by sending inputs to central,
// and providing the output from central asynchronously.
type centralCommunicationImpl struct {
	receiver   CentralReceiver
	sender     CentralSender
	components []common.SensorComponent

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func (s *centralCommunicationImpl) Start(conn *grpc.ClientConn, centralReachable *concurrency.Flag, configHandler config.Handler, detector detector.Detector) {
	go s.sendEvents(central.NewSensorServiceClient(conn), centralReachable, configHandler, detector, s.receiver.Stop, s.sender.Stop)
}

func (s *centralCommunicationImpl) Stop(err error) {
	s.stopC.SignalWithError(err)
}

func (s *centralCommunicationImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

func (s *centralCommunicationImpl) sendEvents(client central.SensorServiceClient, centralReachable *concurrency.Flag, configHandler config.Handler, detector detector.Detector, onStops ...func(error)) {
	defer func() {
		s.stoppedC.SignalWithError(s.stopC.Err())
		runAll(s.stopC.Err(), onStops...)
	}()

	// Start the stream client.
	///////////////////////////
	ctx, err := centralsensor.AppendSensorVersionInfoToContext(context.Background())
	if err != nil {
		s.stopC.SignalWithError(err)
		return
	}

	capsSet := centralsensor.NewSensorCapabilitySet()
	for _, component := range s.components {
		capsSet.AddAll(component.Capabilities()...)
	}
	ctx = centralsensor.AppendCapsInfoToContext(ctx, capsSet)

	stream, err := client.Communicate(ctx)
	if err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "opening stream"))
		return
	}
	_, err = stream.Header()
	if err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "receiving initial metadata"))
		return
	}

	msg, err := stream.Recv()
	if err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "receiving initial cluster config"))
		return
	}

	if msg.GetClusterConfig() == nil {
		s.stopC.SignalWithError(errors.Errorf("initial message received from Sensor was not a cluster config: %T", msg.Msg))
		return
	}

	// Send the initial cluster config to the config handler
	if err := configHandler.ProcessMessage(msg); err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "processing initial cluster config"))
		return
	}

	msg, err = stream.Recv()
	if err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "receiving initial policies"))
		return
	}

	if msg.GetPolicySync() == nil {
		s.stopC.SignalWithError(errors.Errorf("second message received from Sensor was not a policy sync: %T", msg.Msg))
		return
	}

	if err := detector.ProcessMessage(msg); err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "policy sync could not be successfully processed"))
		return
	}

	msg, err = stream.Recv()
	if err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "receiving initial whitelists"))
		return
	}

	// Policy Sync
	if err := detector.ProcessMessage(msg); err != nil {
		s.stopC.SignalWithError(errors.Wrap(err, "process whitelists could not be successfully processed"))
		return
	}

	defer func() {
		if err := stream.CloseSend(); err != nil {
			log.Errorf("Failed to close stream cleanly: %v", err)
		}
	}()
	log.Info("Established connection to Central.")

	centralReachable.Set(true)
	defer centralReachable.Set(false)

	// Start receiving and sending with central.
	////////////////////////////////////////////
	s.receiver.Start(stream, s.Stop, s.sender.Stop)
	s.sender.Start(stream, s.Stop, s.receiver.Stop)
	log.Info("Communication with central started.")

	// Wait for stop.
	/////////////////
	_ = s.stopC.Wait()
	log.Info("Communication with central ended.")
}

func runAll(err error, fs ...func(error)) {
	for _, f := range fs {
		f(err)
	}
}
