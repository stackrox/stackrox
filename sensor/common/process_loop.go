package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/eventstream"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	retryDelay  = 5 * time.Second
	gracePeriod = 5 * time.Second
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

func (s *sensor) sendFlowMessages(flows <-chan *central.NetworkFlowUpdate, flowClient central.NetworkFlowServiceClient) {
	var err error
	recoverable := true
	for !s.stopped.IsDone() && recoverable {
		if err != nil {
			log.Errorf("Recoverable error sending flow updates: %v. Sleeping for %v", err, retryDelay)
			if concurrency.WaitWithTimeout(&s.stopped, retryDelay) {
				break
			}
		}
		recoverable, err = s.sendFlowMessagesSingle(flows, flowClient)
	}
	// Sanity check - if we exit the loop, we should be done, otherwise panic.
	if !concurrency.WaitWithTimeout(&s.stopped, gracePeriod) {
		log.Panicf("Done sending flow updates, but sensor is not stopped. Last error: %v", err)
	}
}

func (s *sensor) sendFlowMessagesSingle(flows <-chan *central.NetworkFlowUpdate, flowClient central.NetworkFlowServiceClient) (recoverable bool, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := flowClient.PushNetworkFlows(ctx)
	if err != nil {
		return true, fmt.Errorf("opening stream: %v", err)
	}
	defer stream.CloseAndRecv()

	for {
		select {
		case flowUpdate, ok := <-flows:
			if !ok {
				return false, errors.New("flows channel closed")
			}
			if err := stream.Send(flowUpdate); err != nil {
				return true, err
			}
		case <-stream.Context().Done():
			return true, stream.Context().Err()
		case <-s.stopped.Done():
			return false, nil
		}
	}
}

func (s *sensor) sendEvents(orchestratorEvents <-chan *v1.SensorEvent, signals <-chan *v1.SensorEvent, output chan<- *v1.SensorEnforcement, eventsClient v1.SensorEventServiceClient) {
	var err error
	recoverable := true
	for !s.stopped.IsDone() && recoverable {
		if err != nil {
			log.Errorf("Recoverable error sending sensor events: %v. Sleeping for %v", err, retryDelay)
			if concurrency.WaitWithTimeout(&s.stopped, retryDelay) {
				break
			}
		}
		recoverable, err = s.sendEventsSingle(orchestratorEvents, signals, output, eventsClient)
	}
	// Sanity check - if we exit the loop, we should be done, otherwise panic.
	if !concurrency.WaitWithTimeout(&s.stopped, gracePeriod) {
		log.Panicf("Done sending sensor events, but sensor is not stopped. Last error: %v", err)
	}
}

func (s *sensor) sendEventsSingle(
	orchestratorEvents <-chan *v1.SensorEvent,
	signals <-chan *v1.SensorEvent,
	output chan<- *v1.SensorEnforcement,
	eventsClient v1.SensorEventServiceClient) (recoverable bool, err error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := eventsClient.RecordEvent(ctx)
	if err != nil {
		return true, fmt.Errorf("opening stream: %v", err)
	}
	defer stream.CloseSend()

	go s.receiveMessages(output, stream)

	eventStream := eventstream.Wrap(stream)
	eventStream = metrics.NewCountingEventStream(eventStream, "unique")
	eventStream = deduper.NewDedupingEventStream(eventStream)
	eventStream = metrics.NewCountingEventStream(eventStream, "total")

	for {
		select {
		case sig, ok := <-signals:
			if !ok {
				return false, errors.New("signals channel closed")
			}
			if err := stream.Send(sig); err != nil {
				return true, err
			}
		case evt, ok := <-orchestratorEvents:
			if !ok {
				return false, errors.New("orchestrator events channel closed")
			}
			if err := eventStream.Send(evt); err != nil {
				return true, err
			}
		case <-stream.Context().Done():
			return true, stream.Context().Err()
		case <-s.stopped.Done():
			log.Infof("Sensor is stopped!")
			return false, nil
		}
	}
}

func (s *sensor) receiveMessages(output chan<- *v1.SensorEnforcement, stream v1.SensorEventService_RecordEventClient) {
	err := s.doReceiveMessages(output, stream)
	if err != nil {
		log.Errorf("Error receiving enforcements from central: %v", err)
	}
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (s *sensor) doReceiveMessages(output chan<- *v1.SensorEnforcement, stream v1.SensorEventService_RecordEventClient) error {
	for {
		select {
		case <-s.stopped.Done():
			return nil
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			// Take in the responses from central and generate enforcements for the outbound channel.
			// Note: Recv blocks until it receives something new, unless the stream closes.
			var eventResp *v1.SensorEnforcement
			eventResp, err := stream.Recv()
			// The loop exits when the stream from central is closed or returns an error.
			if err != nil {
				return err
			}

			// Just to avoid panics, but we currently don't handle any responses not from deployments
			switch x := eventResp.Resource.(type) {
			case *v1.SensorEnforcement_Deployment:
				s.processResponse(stream.Context(), eventResp, output)
			case *v1.SensorEnforcement_ContainerInstance:
				s.processResponse(stream.Context(), eventResp, output)
			default:
				logger.Errorf("Event response with type '%s' is not handled", x)
			}
		}
	}
}

func (s *sensor) processResponse(ctx context.Context, enforcement *v1.SensorEnforcement, output chan<- *v1.SensorEnforcement) {
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
