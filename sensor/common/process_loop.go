package common

import (
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = logging.LoggerForModule()

func logSendingEvent(sensorEvent *v1.SensorEvent) {
	var name string
	var resourceType string
	switch x := sensorEvent.GetResource().(type) {
	case *v1.SensorEvent_Deployment:
		name = sensorEvent.GetDeployment().GetName()
		resourceType = "Deployment"
	case *v1.SensorEvent_NetworkPolicy:
		name = sensorEvent.GetNetworkPolicy().GetName()
		resourceType = "NetworkPolicy"
	case *v1.SensorEvent_Namespace:
		name = sensorEvent.GetNamespace().GetName()
		resourceType = "Namespace"
	case *v1.SensorEvent_ProcessIndicator:
		name = sensorEvent.GetProcessIndicator().GetSignal().GetExecFilePath()
		resourceType = "ProcessIndicator"
	case *v1.SensorEvent_Secret:
		name = sensorEvent.GetSecret().GetName()
		resourceType = "Secret"
	case nil:
		logger.Errorf("Resource field is empty")
	default:
		logger.Errorf("No resource with type %T", x)
	}
	logger.Infof("Sending Sensor Event: Action: '%s'. Type '%s'. Name: '%s'", sensorEvent.GetAction(), resourceType, name)
}

// Take in data from the input channel, pre-process it, then send it to central.
func (s *sensor) sendMessages(
	orchestratorEvents <-chan *v1.SensorEvent,
	signals <-chan *v1.SensorEvent,
	stream v1.SensorEventService_RecordEventClient) {

	// When the input channel closes and looping stops and returns, we need to close the stream with central.
	defer stream.CloseSend()

	for {
		select {
		// Take in events from the inbound channels and send to Central.
		case event, ok := <-orchestratorEvents:
			// The loop stops when the input channel is closed.
			if !ok {
				return
			}
			if event.GetDeployment() != nil {
				s.updateCacheState(event)
			}
			s.sendSensorEventWithLog(stream, event)
		case signal, ok := <-signals:
			if !ok {
				return
			}
			s.sendSensorEventWithLog(stream, signal)
		// If we receive the stop signal, break out of the loop.
		case <-s.stopped.Done():
			return
		}
	}
}

func (s *sensor) updateCacheState(event *v1.SensorEvent) {
	switch event.GetAction() {
	case v1.ResourceAction_CREATE_RESOURCE, v1.ResourceAction_UPDATE_RESOURCE:
		s.enrichImages(event.GetDeployment())
		s.containerCache.AddDeployment(event.GetDeployment())
	case v1.ResourceAction_REMOVE_RESOURCE:
		s.containerCache.RemoveDeployment(event.GetId())
	default:
		logger.Errorf("Resource action not handled: %s", event.GetAction())
		return
	}
}

func (s *sensor) sendSensorEventWithLog(stream v1.SensorEventService_RecordEventClient, sensorEvent *v1.SensorEvent) {
	logSendingEvent(sensorEvent)
	if err := stream.Send(sensorEvent); err != nil {
		log.Errorf("unable to send indicator event: %s", err)
	}
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (s *sensor) receiveMessages(output chan<- *v1.SensorEnforcement, stream v1.SensorEventService_RecordEventClient) {
	var err error
	defer func() { s.Stop(err) }()

	for {
		select {
		case <-s.stopped.Done():
			return

		default:
			// Take in the responses from central and generate enforcements for the outbound channel.
			// Note: Recv blocks until it receives something new, unless the stream closes.
			var eventResp *v1.SensorEnforcement
			eventResp, err = stream.Recv()
			// The loop exits when the stream from central is closed or returns an error.
			if err == io.EOF {
				// Central is not expected to hang up, so we should exit
				// uncleanly to signal the caller to restart.
				log.Info("central hung up (EOF)")
				return
			}
			if s, ok := status.FromError(err); ok && s.Code() == codes.Canceled {
				// The stream has been canceled via its context.
				// Report this as a graceful, intentional termination,
				// instead of an emergent error.
				err = nil
				return
			}
			if err != nil {
				log.Errorf("error reading event response from central: %s", err)
				return
			}

			// Just to avoid panics, but we currently don't handle any responses not from deployments
			switch x := eventResp.Resource.(type) {
			case *v1.SensorEnforcement_Deployment:
				s.processResponse(eventResp, output)
			case *v1.SensorEnforcement_ContainerInstance:
				s.processResponse(eventResp, output)
			default:
				logger.Errorf("Event response with type '%s' is not handled", x)
			}
		}
	}
}

func (s *sensor) processResponse(enforcement *v1.SensorEnforcement, output chan<- *v1.SensorEnforcement) {
	if enforcement == nil {
		return
	}

	if enforcement.GetEnforcement() == v1.EnforcementAction_UNSET_ENFORCEMENT {
		log.Errorf("received enforcement with unset action: %s", proto.MarshalTextString(enforcement))
		if deployment := enforcement.GetDeployment(); deployment != nil {
			log.Infof("deployment processed but no enforcement needed: deployment %s", deployment.GetDeploymentId())
		} else if container := enforcement.GetContainerInstance(); container != nil {
			log.Infof("deployment processed but no enforcement needed: container instance %s", container.GetContainerInstanceId())
		}
		return
	}
	output <- enforcement
}

func (s *sensor) enrichImages(deployment *v1.Deployment) {
	if deployment == nil || len(deployment.GetContainers()) == 0 {
		return
	}
	for _, c := range deployment.GetContainers() {
		s.imageEnricher.EnrichImage(c.Image)
	}
}
