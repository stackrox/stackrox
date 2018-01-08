package listener

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
)

var (
	log = logging.New("listener")
)

// listener provides functionality for listening to deployment events.
type listener struct {
	*dockerClient.Client
	eventsC   chan *v1.DeploymentEvent
	stopC     chan struct{}
	stoppedC  chan struct{}
	clusterID string
}

// New returns a docker listener
func New() (listeners.Listener, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	dockerClient.NegotiateAPIVersion(ctx)
	return &listener{
		Client:    dockerClient,
		eventsC:   make(chan *v1.DeploymentEvent, 10),
		stopC:     make(chan struct{}),
		stoppedC:  make(chan struct{}),
		clusterID: env.ClusterID.Setting(),
	}, nil
}

// Start starts the listener
func (dl *listener) Start() {
	events, errors, cancel := dl.eventHandler()
	dl.sendExistingDeployments()

	log.Info("Swarm Listener Started")
	for {
		select {
		case event := <-events:
			log.Infof("Event: %#v", event)
			dl.pipeDeploymentEvent(event)
		case err := <-errors:
			log.Infof("Reopening stream due to error: %+v", err)
			// Provide a small amount of time for the potential issue to correct itself
			time.Sleep(1 * time.Second)
			events, errors, cancel = dl.eventHandler()
		case <-dl.stopC:
			log.Infof("Shutting down Swarm Listener")
			cancel()
			dl.stoppedC <- struct{}{}
			return
		}
	}
}

func (dl *listener) eventHandler() (<-chan (events.Message), <-chan error, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	filters := filters.NewArgs()
	filters.Add("scope", "swarm")
	filters.Add("type", "service")
	events, errors := dl.Client.Events(ctx, types.EventsOptions{
		Filters: filters,
	})

	return events, errors, cancel
}

func (dl *listener) sendExistingDeployments() {
	existingDeployments, err := dl.getNewExistingDeployments()
	if err != nil {
		log.Errorf("unable to get existing deployments: %s", err)
		return
	}

	for _, d := range existingDeployments {
		dl.eventsC <- &v1.DeploymentEvent{
			Deployment: d,
			Action:     v1.ResourceAction_CREATE_RESOURCE,
			ClusterId:  dl.clusterID,
		}

	}
}

func (dl *listener) getNewExistingDeployments() ([]*v1.Deployment, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	swarmServices, err := dl.Client.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	deployments := make([]*v1.Deployment, len(swarmServices))
	for i, service := range swarmServices {
		d := serviceWrap(service).asDeployment(dl.Client)
		deployments[i] = d
	}
	return deployments, nil
}

func (dl *listener) getDeploymentFromServiceID(id string) (*v1.Deployment, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	serviceInfo, _, err := dl.Client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
	if err != nil {
		return nil, err
	}
	return serviceWrap(serviceInfo).asDeployment(dl.Client), nil
}

func (dl *listener) pipeDeploymentEvent(msg events.Message) {
	actor := msg.Type
	id := msg.Actor.ID

	var resourceAction v1.ResourceAction
	var deployment *v1.Deployment
	var err error

	switch msg.Action {
	case "create":
		resourceAction = v1.ResourceAction_CREATE_RESOURCE

		if deployment, err = dl.getDeploymentFromServiceID(id); err != nil {
			log.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "update":
		resourceAction = v1.ResourceAction_UPDATE_RESOURCE

		if deployment, err = dl.getDeploymentFromServiceID(id); err != nil {
			log.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "remove":
		resourceAction = v1.ResourceAction_REMOVE_RESOURCE

		deployment = &v1.Deployment{
			Id: id,
		}
	default:
		resourceAction = v1.ResourceAction_UNSET_ACTION_RESOURCE
		log.Warnf("unknown action: %s", msg.Action)
		return
	}

	event := &v1.DeploymentEvent{
		Deployment: deployment,
		Action:     resourceAction,
		ClusterId:  dl.clusterID,
	}

	dl.eventsC <- event
}

// Events is the mechanism through which the events are propagated back to the event loop
func (dl *listener) Events() <-chan *v1.DeploymentEvent {
	return dl.eventsC
}

func (dl *listener) Stop() {
	dl.stopC <- struct{}{}
	<-dl.stoppedC
}
