package swarm

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/listeners"
	listenerTypes "bitbucket.org/stack-rox/apollo/apollo/listeners/types"
	apolloTypes "bitbucket.org/stack-rox/apollo/apollo/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
)

var (
	log = logging.New("listener/swarm")
)

func init() {
	listeners.Registry["swarm"] = func(storage db.DeploymentStorage) (listenerTypes.Listener, error) {
		d, err := New(storage)
		return d, err
	}
}

// listener provides functionality for listening to deployment events.
type listener struct {
	*dockerClient.Client
	eventsC  chan apolloTypes.DeploymentEvent
	stopC    chan struct{}
	stoppedC chan struct{}
	storage  db.DeploymentStorage
}

// New returns a docker listener
func New(storage db.DeploymentStorage) (listenerTypes.Listener, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	if err := docker.NegotiateClientVersionToLatest(dockerClient, docker.DefaultAPIVersion); err != nil {
		return nil, err
	}
	return &listener{
		Client:   dockerClient,
		eventsC:  make(chan apolloTypes.DeploymentEvent, 10),
		stopC:    make(chan struct{}),
		stoppedC: make(chan struct{}),
		storage:  storage,
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
		dl.eventsC <- apolloTypes.DeploymentEvent{
			Deployment: d,
			Action:     apolloTypes.Create,
		}

		if err = dl.storage.AddDeployment(d); err != nil {
			log.Errorf("unable to add deployment %s: %s", d.GetId(), err)
		}
	}
}

func (dl *listener) getNewExistingDeployments() ([]*v1.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), docker.HangTimeout)
	defer cancel()

	swarmServices, err := dl.Client.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	deployments := make([]*v1.Deployment, len(swarmServices))
	for i, service := range swarmServices {
		d := serviceWrap(service).asDeployment()
		deployments[i] = d
	}
	newDeployments := dl.filterKnownDeployments(deployments)
	return newDeployments, nil
}

func (dl *listener) filterKnownDeployments(deployments []*v1.Deployment) (output []*v1.Deployment) {
	for _, d := range deployments {
		if saved, exists, err := dl.storage.GetDeployment(d.GetId()); err != nil || !exists || saved.GetVersion() != d.GetVersion() {
			output = append(output, d)
		}
	}

	return
}

func (dl *listener) getDeploymentFromServiceID(id string) (*v1.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), docker.HangTimeout)
	defer cancel()

	serviceInfo, _, err := dl.Client.ServiceInspectWithRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	return serviceWrap(serviceInfo).asDeployment(), nil
}

func (dl *listener) pipeDeploymentEvent(msg events.Message) {
	actor := msg.Type
	id := msg.Actor.ID

	var resourceAction apolloTypes.ResourceAction
	var deployment *v1.Deployment
	var err error

	switch msg.Action {
	case "create":
		resourceAction = apolloTypes.Create

		if deployment, err = dl.getDeploymentFromServiceID(id); err != nil {
			log.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}

		if err = dl.storage.AddDeployment(deployment); err != nil {
			log.Errorf("unable to add deployment %s: %s", deployment.GetId(), err)
		}
	case "update":
		resourceAction = apolloTypes.Update

		if deployment, err = dl.getDeploymentFromServiceID(id); err != nil {
			log.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}

		if err = dl.storage.UpdateDeployment(deployment); err != nil {
			log.Errorf("unable to update deployment %s: %s", deployment.GetId(), err)
		}
	case "remove":
		resourceAction = apolloTypes.Remove

		if err = dl.storage.RemoveDeployment(deployment.GetId()); err != nil {
			log.Errorf("unable to remove deployment %s: %s", deployment.GetId(), err)
		}

		deployment = &v1.Deployment{
			Id: id,
		}
	default:
		resourceAction = apolloTypes.Unknown
		log.Warnf("unknown action: %s", msg.Action)
		return
	}

	event := apolloTypes.DeploymentEvent{
		Deployment: deployment,
		Action:     resourceAction,
	}

	dl.eventsC <- event
}

// Events is the mechanism through which the events are propagated back to the event loop
func (dl *listener) Events() <-chan apolloTypes.DeploymentEvent {
	return dl.eventsC
}

func (dl *listener) Stop() {
	dl.stopC <- struct{}{}
	<-dl.stoppedC
}
