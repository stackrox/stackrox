package sensor

import (
	"io"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/enforcers"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
)

// newProcessLoops returns a new Sensor.
func newProcessLoops(imageIntegrationPoller *sources.Client) *processLoopsImpl {
	return &processLoopsImpl{
		pendingCache:     newPendingDeployments(),
		eventWrapToEvent: eventWrapToEvent(imageIntegrationPoller),
	}
}

// processLoopsImpl is the master processing logic underlying Sensor. It consumes sensor's inputs and returns
// it's outputs.
type processLoopsImpl struct {
	pendingCache     *pendingDeploymentEvents
	eventWrapToEvent func(*listeners.DeploymentEventWrap) (*v1.DeploymentEvent, error)

	stopLoop    chan struct{}
	loopStopped chan struct{}
}

// Starts the processing loops.
func (p *processLoopsImpl) startLoops(input <-chan *listeners.DeploymentEventWrap, stream v1.SensorEventService_RecordEventClient, output chan<- *enforcers.DeploymentEnforcement) {
	p.stopLoop = make(chan struct{})
	p.loopStopped = make(chan struct{})

	go p.sendMessages(input, stream)
	go p.receiveMessages(stream, output)
}

// Stops the processing loops.
func (p *processLoopsImpl) stopLoops() {
	p.stopLoop <- struct{}{}
	<-p.loopStopped

	close(p.stopLoop)
	close(p.loopStopped)
}

// The processing loops which feed the input channel data to central,
// and returns the data output from central to the output channel.
//////////////////////////////////////////////////////////////////

// Take in data from the input channel, pre-process it, then send it to central.
func (p *processLoopsImpl) sendMessages(input <-chan *listeners.DeploymentEventWrap, stream v1.SensorEventService_RecordEventClient) {
	// When the input channel closes and looping stops and returns, we need to close the stream with central.
	defer stream.CloseSend()

	for {
		select {
		// Take in events from the inbound channel, pre-process, then send to central.
		case eventWrap, ok := <-input:
			// The loop stops when the input channel is closed.
			if !ok {
				return
			}

			alreadyPresent := p.pendingCache.add(eventWrap)
			if alreadyPresent && eventWrap.GetAction() != v1.ResourceAction_REMOVE_RESOURCE {
				continue
			} else if alreadyPresent && eventWrap.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
				p.pendingCache.remove(eventWrap)
			}

			event, err := p.eventWrapToEvent(eventWrap)
			if err != nil {
				log.Errorf("unable to handle event wrapper: %s", err)
				continue
			}

			log.Infof("deployment already being processed %s", eventWrap.GetDeployment().GetId())
			if err := stream.Send(event); err != nil {
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

		if eventResp.GetEnforcement() == v1.EnforcementAction_UNSET_ENFORCEMENT {
			log.Infof("deployment processed but no enforcement needed %s", eventResp.GetDeploymentId())
			continue
		}

		log.Infof("enforcement requested for deployment %s", eventResp.GetDeploymentId())
		eventWrap, exists := p.pendingCache.fetch(eventResp.GetDeploymentId())
		if !exists {
			log.Errorf("cannot find deployment event for deployment %s", eventResp.GetDeploymentId())
			continue
		}

		log.Infof("performing enforcement %s on deployment %s", eventWrap.GetAction(), eventWrap.GetDeployment().GetName())
		output <- &enforcers.DeploymentEnforcement{
			Deployment:   eventWrap.GetDeployment(),
			OriginalSpec: eventWrap.OriginalSpec,
			Enforcement:  eventResp.GetEnforcement(),
			AlertID:      eventResp.GetAlertId(),
		}
	}
}
