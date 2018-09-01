package sensor

import (
	"io"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
	sensorMetrics "github.com/stackrox/rox/pkg/sensor/metrics"
	"google.golang.org/grpc"
)

// newProcessLoops returns a new Sensor.
func newProcessLoops(centralConn *grpc.ClientConn, clusterID string) *processLoopsImpl {
	// This will track the set of integrations for this cluster.
	integrationSet := integration.NewSet()

	// This polls central for the integrations specific to this cluster.
	poller := integration.NewPoller(integrationSet, centralConn, clusterID)

	// This uses those integrations to enrich images.
	imageEnricher := enricher.New(integrationSet, metrics.SensorSubsystem)

	return &processLoopsImpl{
		imageEnricher: imageEnricher,
		poller:        poller,
		pendingCache:  newPendingEvents(),
	}
}

// processLoopsImpl is the master processing logic underlying Sensor. It consumes sensor's inputs and returns
// it's outputs.
type processLoopsImpl struct {
	pendingCache  *pendingEvents
	imageEnricher enricher.ImageEnricher
	poller        integration.Poller

	stopLoop    concurrency.Signal
	loopStopped concurrency.Signal
}

// Starts the processing loops.
func (p *processLoopsImpl) startLoops(orchestratorInput <-chan *listeners.EventWrap,
	collectorInput <-chan *listeners.EventWrap,
	stream v1.SensorEventService_RecordEventClient,
	output chan<- *enforcers.DeploymentEnforcement) {

	go p.poller.Start()

	p.stopLoop = concurrency.NewSignal()
	p.loopStopped = concurrency.NewSignal()

	go p.sendMessages(orchestratorInput, collectorInput, stream)
	go p.receiveMessages(stream, output)
}

// Stops the processing loops.
func (p *processLoopsImpl) stopLoops() {
	p.poller.Stop()

	p.stopLoop.Signal()
	p.loopStopped.Wait()
}

// The processing loops which feed the input channel data to central,
// and returns the data output from central to the output channel.
//////////////////////////////////////////////////////////////////

func logSendingEvent(eventWrap *listeners.EventWrap) {
	var name string
	var resourceType string
	switch x := eventWrap.Resource.(type) {
	case *v1.SensorEvent_Deployment:
		name = eventWrap.GetDeployment().GetName()
		resourceType = "Deployment"
	case *v1.SensorEvent_NetworkPolicy:
		name = eventWrap.GetNetworkPolicy().GetName()
		resourceType = "NetworkPolicy+"
	case *v1.SensorEvent_Namespace:
		name = eventWrap.GetNamespace().GetName()
		resourceType = "Namespace"
	case *v1.SensorEvent_Indicator:
		name = eventWrap.GetIndicator().GetId()
		resourceType = "Indicator"
	case *v1.SensorEvent_Secret:
		name = eventWrap.GetSecret().GetName()
		resourceType = "Secret"
	case nil:
		logger.Errorf("Resource field is empty")
	default:
		logger.Errorf("No resource with type %T", x)
	}
	logger.Infof("Sending Sensor Event: Action: '%s'. Type '%s'. Name: '%s'", eventWrap.GetAction(), resourceType, name)
}

// Take in data from the input channel, pre-process it, then send it to central.
func (p *processLoopsImpl) sendMessages(orchestratorInput <-chan *listeners.EventWrap,
	collectorInput <-chan *listeners.EventWrap,
	stream v1.SensorEventService_RecordEventClient) {

	// When the input channel closes and looping stops and returns, we need to close the stream with central.
	defer stream.CloseSend()

	for {
		select {
		// Take in events from the inbound channel, pre-process, then send to central.
		case eventWrap, ok := <-orchestratorInput:
			// The loop stops when the input channel is closed.
			if !ok {
				return
			}

			// exactMatchInCache implies that the data has changed in the orchestrator, but is not tracked by
			// our objects so we can ignore
			switch eventWrap.GetAction() {
			case v1.ResourceAction_REMOVE_RESOURCE:
				p.pendingCache.remove(eventWrap)
			case v1.ResourceAction_CREATE_RESOURCE, v1.ResourceAction_UPDATE_RESOURCE:
				if exactMatchInCache := p.pendingCache.add(eventWrap); exactMatchInCache {
					continue
				}
				p.enrichImages(eventWrap.GetDeployment())
			default:
				logger.Errorf("Resource action not handled: %s", eventWrap.GetAction())
				continue
			}

			logSendingEvent(eventWrap)
			if err := stream.Send(eventWrap.SensorEvent); err != nil {
				log.Errorf("unable to send orchestrator event: %s", err)
			}
		case eventWrap, ok := <-collectorInput:
			if !ok {
				return
			}
			indicator, ok := eventWrap.GetResource().(*v1.SensorEvent_Indicator)
			if !ok {
				log.Errorf("Non indicator SensorEvent found on collector Input channel: %v", eventWrap)
				continue
			}
			indicator.Indicator.EmitTimestamp = types.TimestampNow()
			lag := time.Now().Sub(protoconv.ConvertTimestampToTimeOrNow(indicator.Indicator.GetSignal().GetTime()))
			sensorMetrics.RegisterSignalToIndicatorEmitLag(eventWrap.GetClusterId(), float64(lag))
			if err := stream.Send(eventWrap.SensorEvent); err != nil {
				log.Errorf("unable to send indicator event: %s", err)
			}
		// If we receive the stop signal, then break out of the loop.
		case <-p.stopLoop.Done():
			return
		}
	}
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (p *processLoopsImpl) receiveMessages(stream v1.SensorEventService_RecordEventClient, output chan<- *enforcers.DeploymentEnforcement) {
	defer p.loopStopped.Signal()

	for {
		// Take in the responses from central and generate enforcements for the outbound channel.
		eventResp, err := stream.Recv()
		// The loop stops when the stream from central is closed or returns an error.
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Errorf("error reading event response from central: %s", err)
			return
		}

		// Just to avoid panics, but we currently don't handle any responses not from deployments
		switch x := eventResp.Resource.(type) {
		case *v1.SensorEventResponse_Deployment:
			p.processDeploymentResponse(eventResp, output)
		case *v1.SensorEventResponse_NetworkPolicy, *v1.SensorEventResponse_Namespace, *v1.SensorEventResponse_Indicator, *v1.SensorEventResponse_Secret:
			// Purposefully eating the responses for these types because there is nothing to do for them
		default:
			logger.Errorf("Event response with type '%s' is not handled", x)
		}
	}
}

func (p *processLoopsImpl) processDeploymentResponse(eventResp *v1.SensorEventResponse, output chan<- *enforcers.DeploymentEnforcement) {
	deploymentResp := eventResp.GetDeployment()
	eventWrap, exists := p.pendingCache.fetch(deploymentResp.GetDeploymentId())
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

func (p *processLoopsImpl) enrichImages(deployment *v1.Deployment) {
	if deployment == nil || len(deployment.GetContainers()) == 0 {
		return
	}
	for _, c := range deployment.GetContainers() {
		p.imageEnricher.EnrichImage(c.Image)
	}
}
