package sensor

import (
	"io"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sensor/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	signalRetries       = 10
	signalRetryInterval = 2 * time.Second
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
		name = sensorEvent.GetProcessIndicator().GetSignal().GetProcessSignal().GetCommandLine()
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

func (s *sensor) reprocessSignalLater(stream v1.SensorEventService_RecordEventClient, sensorEvent *v1.SensorEvent) {
	t := time.NewTicker(signalRetryInterval)
	logger.Infof("Trying to reprocess '%s'", sensorEvent.GetProcessIndicator().GetSignal().GetProcessSignal().GetCommandLine())
	indicator := sensorEvent.GetProcessIndicator()
	for i := 0; i < signalRetries; i++ {
		<-t.C
		deploymentID, exists := s.pendingCache.FetchDeploymentByContainer(indicator.GetSignal().GetContainerId())
		if exists {
			indicator.DeploymentId = deploymentID
			s.sendIndicatorEvent(stream, sensorEvent)
			return
		}
	}
	logger.Errorf("Dropping this on the floor: %+v", proto.MarshalTextString(indicator))
}

// Take in data from the input channel, pre-process it, then send it to central.
func (s *sensor) sendMessages(
	deployments <-chan *listeners.EventWrap,
	signals <-chan *listeners.EventWrap,
	stream v1.SensorEventService_RecordEventClient) {

	// When the input channel closes and looping stops and returns, we need to close the stream with central.
	defer stream.CloseSend()

	for {
		select {
		// Take in events from the inbound channel, pre-process, then send to central.
		case deploy, ok := <-deployments:
			// The loop stops when the input channel is closed.
			if !ok {
				return
			}
			s.sendDeploy(deploy, stream)

		case signal, ok := <-signals:
			if !ok {
				return
			}
			s.sendSignal(signal, stream)

		// If we receive the stop signal, break out of the loop.
		case <-s.stopped.Done():
			return
		}
	}
}

func (s *sensor) sendDeploy(eventWrap *listeners.EventWrap, stream v1.SensorEventService_RecordEventClient) {
	switch eventWrap.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		s.pendingCache.RemoveDeployment(eventWrap)
	case v1.ResourceAction_CREATE_RESOURCE, v1.ResourceAction_UPDATE_RESOURCE:
		// Not adding the event implies that it already exists in its exact form in the cache.
		if eventAdded := s.pendingCache.AddDeployment(eventWrap); !eventAdded {
			return
		}
		s.enrichImages(eventWrap.GetDeployment())
	default:
		logger.Errorf("Resource action not handled: %s", eventWrap.GetAction())
		return
	}
	s.sendSensorEventWithLog(stream, eventWrap.SensorEvent)
}

func (s *sensor) sendSignal(eventWrap *listeners.EventWrap, stream v1.SensorEventService_RecordEventClient) {
	indicatorWrap, ok := eventWrap.GetResource().(*v1.SensorEvent_ProcessIndicator)
	if !ok {
		log.Errorf("Non-indicator SensorEvent found on collector input channel: %v", eventWrap)
		return
	}
	indicator := indicatorWrap.ProcessIndicator

	// populate deployment id
	deploymentID, exists := s.pendingCache.FetchDeploymentByContainer(indicator.GetSignal().GetContainerId())
	if !exists {
		go s.reprocessSignalLater(stream, eventWrap.SensorEvent)
		return
	}
	indicator.DeploymentId = deploymentID
	s.sendIndicatorEvent(stream, eventWrap.SensorEvent)
}

func (s *sensor) sendIndicatorEvent(stream v1.SensorEventService_RecordEventClient, sensorEvent *v1.SensorEvent) {
	sensorEvent.Resource.(*v1.SensorEvent_ProcessIndicator).ProcessIndicator.EmitTimestamp = types.TimestampNow()
	lag := time.Now().Sub(protoconv.ConvertTimestampToTimeOrNow(sensorEvent.GetProcessIndicator().GetSignal().GetTime()))
	metrics.RegisterSignalToIndicatorEmitLag(sensorEvent.GetClusterId(), float64(lag))
	s.sendSensorEventWithLog(stream, sensorEvent)
}

func (s *sensor) sendSensorEventWithLog(stream v1.SensorEventService_RecordEventClient, sensorEvent *v1.SensorEvent) {
	logSendingEvent(sensorEvent)
	if err := stream.Send(sensorEvent); err != nil {
		log.Errorf("unable to send indicator event: %s", err)
	}
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (s *sensor) receiveMessages(output chan<- *enforcers.DeploymentEnforcement, stream v1.SensorEventService_RecordEventClient) {
	var err error
	defer func() { s.Stop(err) }()

	for {
		select {
		case <-s.stopped.Done():
			return

		default:
			// Take in the responses from central and generate enforcements for the outbound channel.
			// Note: Recv blocks until it receives something new, unless the stream closes.
			var eventResp *v1.SensorEventResponse
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
			case *v1.SensorEventResponse_Deployment:
				s.processDeploymentResponse(eventResp, output)
			case *v1.SensorEventResponse_NetworkPolicy:
			case *v1.SensorEventResponse_Namespace:
			case *v1.SensorEventResponse_Indicator:
			case *v1.SensorEventResponse_Secret:
				// Drop the values for these types because there is nothing to do for them.
			default:
				logger.Errorf("Event response with type '%s' is not handled", x)
			}
		}
	}
}

func (s *sensor) processDeploymentResponse(eventResp *v1.SensorEventResponse, output chan<- *enforcers.DeploymentEnforcement) {
	deploymentResp := eventResp.GetDeployment()
	eventWrap, exists := s.pendingCache.FetchDeployment(deploymentResp.GetDeploymentId())
	if !exists {
		log.Errorf("cannot find deployment event for deployment %s", deploymentResp.GetDeploymentId())
		return
	}

	if deploymentResp.GetEnforcement() == v1.EnforcementAction_UNSET_ENFORCEMENT {
		log.Infof("deployment processed but no enforcement needed on %s", eventWrap.GetDeployment().GetName())
		return
	}

	log.Infof("enforcement requested for deployment %s", deploymentResp.GetDeploymentId())

	log.Infof("performing enforcement %s on deployment %s", eventWrap.GetAction(), eventWrap.GetDeployment().GetName())
	output <- &enforcers.DeploymentEnforcement{
		Deployment:   eventWrap.GetDeployment(),
		OriginalSpec: eventWrap.OriginalSpec,
		Enforcement:  deploymentResp.GetEnforcement(),
		AlertID:      deploymentResp.GetAlertId(),
	}
}

func (s *sensor) enrichImages(deployment *v1.Deployment) {
	if deployment == nil || len(deployment.GetContainers()) == 0 {
		return
	}
	for _, c := range deployment.GetContainers() {
		s.imageEnricher.EnrichImage(c.Image)
	}
}
