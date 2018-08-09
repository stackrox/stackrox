package sensor

import (
	"io"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/listeners"
	"google.golang.org/grpc"
)

// newProcessLoops returns a new Sensor.
func newProcessLoops(centralConn *grpc.ClientConn, clusterID string) *processLoopsImpl {
	// This will track the set of integrations for this cluster.
	integrationSet := integration.NewSet()

	// This polls central for the integrations specific to this cluster.
	poller := integration.NewPoller(integrationSet, centralConn, clusterID)

	// This uses those integrations to enrich images.
	imageEnricher := enricher.New(integrationSet)

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

	stopLoop    chan struct{}
	loopStopped chan struct{}
}

// Starts the processing loops.
func (p *processLoopsImpl) startLoops(input <-chan *listeners.EventWrap, stream v1.SensorEventService_RecordEventClient, output chan<- *enforcers.DeploymentEnforcement) {
	go p.poller.Start()

	p.stopLoop = make(chan struct{})
	p.loopStopped = make(chan struct{})
	go p.sendMessages(input, stream)
	go p.receiveMessages(stream, output)
}

// Stops the processing loops.
func (p *processLoopsImpl) stopLoops() {
	p.poller.Stop()

	p.stopLoop <- struct{}{}
	<-p.loopStopped
	close(p.stopLoop)
	close(p.loopStopped)
}

// The processing loops which feed the input channel data to central,
// and returns the data output from central to the output channel.
//////////////////////////////////////////////////////////////////

// Take in data from the input channel, pre-process it, then send it to central.
func (p *processLoopsImpl) sendMessages(eventInput <-chan *listeners.EventWrap, stream v1.SensorEventService_RecordEventClient) {
	// When the input channel closes and looping stops and returns, we need to close the stream with central.
	defer stream.CloseSend()

	for {
		select {
		// Take in events from the inbound channel, pre-process, then send to central.
		case eventWrap, ok := <-eventInput:
			// The loop stops when the input channel is closed.
			if !ok {
				return
			}

			alreadyPresent := p.pendingCache.add(eventWrap)
			if alreadyPresent && eventWrap.GetAction() != v1.ResourceAction_REMOVE_RESOURCE {
				continue
			} else if alreadyPresent && eventWrap.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
				p.pendingCache.remove(eventWrap)
			} else if !alreadyPresent && eventWrap.GetAction() != v1.ResourceAction_REMOVE_RESOURCE {
				p.enrichImages(eventWrap.GetDeployment())
			}

			log.Infof("Event already being processed %s", eventWrap.GetId())
			if err := stream.Send(eventWrap.SensorEvent); err != nil {
				log.Errorf("unable to send event: %s", err)
			}

		// If we receive the stop signal, then break out of the loop.
		case _ = <-p.stopLoop:
			break
		}
	}
}

// Take in data processed by central, run post processing, then send it to the output channel.
func (p *processLoopsImpl) receiveMessages(stream v1.SensorEventService_RecordEventClient, output chan<- *enforcers.DeploymentEnforcement) {
	defer func() { p.loopStopped <- struct{}{} }()

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
		case *v1.SensorEventResponse_NetworkPolicy, *v1.SensorEventResponse_Namespace:
			// Purposefully eating the responses for these types because there is nothing to do for them
		default:
			logger.Errorf("Event response with type '%s' is not handled", x)
		}
	}
}

func (p *processLoopsImpl) processDeploymentResponse(eventResp *v1.SensorEventResponse, output chan<- *enforcers.DeploymentEnforcement) {
	deploymentResp := eventResp.GetDeployment()
	if deploymentResp.GetEnforcement() == v1.EnforcementAction_UNSET_ENFORCEMENT {
		log.Infof("deployment processed but no enforcement needed %s", deploymentResp.GetDeploymentId())
		return
	}

	log.Infof("enforcement requested for deployment %s", deploymentResp.GetDeploymentId())
	eventWrap, exists := p.pendingCache.fetch(deploymentResp.GetDeploymentId())
	if !exists {
		log.Errorf("cannot find deployment event for deployment %s", deploymentResp.GetDeploymentId())
		return
	}

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
