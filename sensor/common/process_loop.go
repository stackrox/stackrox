package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/messagestream"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	retryDelay  = 5 * time.Second
	gracePeriod = 5 * time.Second
)

var logger = logging.LoggerForModule()

func logSendingEvent(sensorEvent *central.SensorEvent) {
	var name string
	var resourceType string
	switch x := sensorEvent.GetResource().(type) {
	case *central.SensorEvent_Deployment:
		name = sensorEvent.GetDeployment().GetName()
		resourceType = "Deployment"
	case *central.SensorEvent_NetworkPolicy:
		name = sensorEvent.GetNetworkPolicy().GetName()
		resourceType = "NetworkPolicy"
	case *central.SensorEvent_Namespace:
		name = sensorEvent.GetNamespace().GetName()
		resourceType = "Namespace"
	case *central.SensorEvent_ProcessIndicator:
		name = sensorEvent.GetProcessIndicator().GetSignal().GetExecFilePath()
		resourceType = "ProcessIndicator"
	case *central.SensorEvent_Secret:
		name = sensorEvent.GetSecret().GetName()
		resourceType = "Secret"
	case nil:
		logger.Errorf("Resource field is empty")
	default:
		logger.Errorf("No resource with type %T", x)
	}
	logger.Infof("Sending Sensor Event: Action: '%s'. Type '%s'. Name: '%s'", sensorEvent.GetAction(), resourceType, name)
}

func (s *sensor) sendEvents(orchestratorEvents <-chan *central.SensorEvent, signals <-chan *central.SensorEvent, flows <-chan *central.NetworkFlowUpdate, output chan<- *central.SensorEnforcement, client central.SensorServiceClient) {
	var err error
	recoverable := true
	for !s.stopped.IsDone() && recoverable {
		if err != nil {
			log.Errorf("Recoverable error sending sensor events: %v. Sleeping for %v", err, retryDelay)
			if concurrency.WaitWithTimeout(&s.stopped, retryDelay) {
				break
			}
		}
		recoverable, err = s.sendEventsSingle(orchestratorEvents, signals, flows, output, client)
	}
	// Sanity check - if we exit the loop, we should be done, otherwise panic.
	if !concurrency.WaitWithTimeout(&s.stopped, gracePeriod) {
		log.Panicf("Done sending sensor events, but sensor is not stopped. Last error: %v", err)
	}
}

func (s *sensor) sendEventsSingle(
	orchestratorEvents <-chan *central.SensorEvent,
	signals <-chan *central.SensorEvent,
	flows <-chan *central.NetworkFlowUpdate,
	output chan<- *central.SensorEnforcement,
	client central.SensorServiceClient) (recoverable bool, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Communicate(ctx)
	if err != nil {
		return true, fmt.Errorf("opening stream: %v", err)
	}
	defer stream.CloseSend()

	go s.receiveMessages(output, stream)

	wrappedStream := messagestream.Wrap(stream)
	wrappedStream = metrics.NewCountingEventStream(wrappedStream, "unique")
	wrappedStream = deduper.NewDedupingMessageStream(wrappedStream)
	wrappedStream = metrics.NewCountingEventStream(wrappedStream, "total")

	for {
		var msg *central.MsgFromSensor
		select {
		case sig, ok := <-signals:
			if !ok {
				return false, errors.New("signals channel closed")
			}
			msg = &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: sig,
				},
			}
		case evt, ok := <-orchestratorEvents:
			if !ok {
				return false, errors.New("orchestrator events channel closed")
			}
			msg = &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: evt,
				},
			}
		case flowUpdate, ok := <-flows:
			if !ok {
				return false, errors.New("flow updates channel closed")
			}
			msg = &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_NetworkFlowUpdate{
					NetworkFlowUpdate: flowUpdate,
				},
			}
		case <-stream.Context().Done():
			return true, stream.Context().Err()
		case <-s.stopped.Done():
			log.Infof("Sensor is stopped!")
			return false, nil
		}

		if msg != nil {
			if err := stream.Send(msg); err != nil {
				return true, err
			}
		}
	}
}

func (s *sensor) receiveMessages(output chan<- *central.SensorEnforcement, stream central.SensorService_CommunicateClient) {
	err := s.doReceiveMessages(output, stream)
	if err != nil {
		log.Errorf("Error receiving enforcements from central: %v", err)
	}
	s.stopped.SignalWithError(err)
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (s *sensor) doReceiveMessages(output chan<- *central.SensorEnforcement, stream central.SensorService_CommunicateClient) error {
	for {
		select {
		case <-s.stopped.Done():
			return s.stopped.Err()
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			// Take in the responses from central and generate enforcements for the outbound channel.
			// Note: Recv blocks until it receives something new, unless the stream closes.
			msg, err := stream.Recv()
			// The loop exits when the stream from central is closed or returns an error.
			if err != nil {
				return err
			}

			enforcementMsg, ok := msg.Msg.(*central.MsgToSensor_Enforcement)
			if !ok {
				logger.Errorf("Unsupported message from central of type %T: %+v", msg.Msg, msg.Msg)
				continue
			}
			enforcement := enforcementMsg.Enforcement

			switch x := enforcement.Resource.(type) {
			case *central.SensorEnforcement_Deployment:
				s.processResponse(stream.Context(), enforcement, output)
			case *central.SensorEnforcement_ContainerInstance:
				s.processResponse(stream.Context(), enforcement, output)
			default:
				logger.Errorf("Event response with type '%s' is not handled", x)
			}
		}
	}
}

func (s *sensor) processResponse(ctx context.Context, enforcement *central.SensorEnforcement, output chan<- *central.SensorEnforcement) {
	if enforcement == nil {
		return
	}

	if enforcement.GetEnforcement() == storage.EnforcementAction_UNSET_ENFORCEMENT {
		log.Errorf("received enforcement with unset action: %s", proto.MarshalTextString(enforcement))
		if deployment := enforcement.GetDeployment(); deployment != nil {
			log.Infof("deployment processed but no enforcement needed: deployment %s", deployment.GetDeploymentId())
		} else if container := enforcement.GetContainerInstance(); container != nil {
			log.Infof("deployment processed but no enforcement needed: container instance %s", container.GetContainerInstanceId())
		}
		return
	}

	select {
	case output <- enforcement:
	case <-s.stopped.Done():
	case <-ctx.Done():
	}
}

func (s *sensor) enrichImages(deployment *storage.Deployment) {
	if deployment == nil || len(deployment.GetContainers()) == 0 {
		return
	}
	for _, c := range deployment.GetContainers() {
		s.imageEnricher.EnrichImage(c.Image)
	}
}
