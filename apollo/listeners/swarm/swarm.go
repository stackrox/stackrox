package swarm

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/listeners"
	listenerTypes "bitbucket.org/stack-rox/apollo/apollo/listeners/types"
	apolloTypes "bitbucket.org/stack-rox/apollo/apollo/types"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

var (
	log = logging.New("listener/swarm")
)

// Listener is a wrapper around the docker client which institutes a container cache
type Listener struct {
	*dockerClient.Client
	eventsChan chan apolloTypes.Event
	done       chan struct{}
	finished   chan struct{}

	resourceCache map[string]*apolloTypes.Container // resourceID to resource
}

// New returns a docker listener
func New() (*Listener, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	if err := docker.NegotiateClientVersionToLatest(dockerClient, docker.DefaultAPIVersion); err != nil {
		return nil, err
	}
	return &Listener{
		dockerClient,
		make(chan apolloTypes.Event),
		make(chan struct{}),
		make(chan struct{}),
		make(map[string]*apolloTypes.Container),
	}, nil
}

// Done is called before the program exits
func (dl *Listener) Done() {
	dl.done <- struct{}{}
	<-dl.finished
}

func (dl *Listener) getContainers() ([]*apolloTypes.Container, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	// Currently no filters until we see what shows up. Maybe filter out UCP?
	var slo types.ServiceListOptions
	swarmServices, err := dl.Client.ServiceList(ctx, slo)
	if err != nil {
		return nil, err
	}
	containers := make([]*apolloTypes.Container, len(swarmServices))
	for i, service := range swarmServices {
		container := swarmToContainer{service: service}.ConvertToContainer()
		containers[i] = container
		dl.resourceCache[container.ID] = container
	}
	return containers, nil
}

func (dl *Listener) getContainer(id string) (*apolloTypes.Container, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	serviceInfo, _, err := dl.Client.ServiceInspectWithRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	return swarmToContainer{service: serviceInfo}.ConvertToContainer(), nil
}

// GetContainers returns all of the currently running containers
func (dl *Listener) GetContainers() ([]*apolloTypes.Container, error) {
	var resources []*apolloTypes.Container
	services, err := dl.getContainers()
	if err != nil {
		return nil, err
	}
	resources = append(resources, services...)
	return resources, nil
}

// Start starts the listener
func (dl *Listener) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	filters := filters.NewArgs()
	filters.Add("scope", "swarm")
	events, errors := dl.Client.Events(ctx, types.EventsOptions{
		Filters: filters,
	})
	for {
		select {
		case event := <-events:
			dl.processEvent(event)
		case err := <-errors:
			log.Infof("Reopening stream due above error: %+v", err)
			// Provide a small amount of time for the potential issue to correct itself
			time.Sleep(1 * time.Second)
			events, errors = dl.Client.Events(ctx, types.EventsOptions{})
		case <-dl.done:
			log.Infof("Shutting down docker listener")
			cancel()
			dl.finished <- struct{}{}
			return
		}
	}
}

// Events is the mechanism through which the events are propagated back to the event loop
func (dl *Listener) Events() <-chan apolloTypes.Event {
	return dl.eventsChan
}

func (dl *Listener) processEvent(msg events.Message) {
	actor := msg.Type
	id := msg.Actor.ID

	var resourceAction apolloTypes.ResourceAction
	switch msg.Action {
	case "create":
		resourceAction = apolloTypes.Create
	case "remove":
		resourceAction = apolloTypes.Remove
	case "update":
		resourceAction = apolloTypes.Update
	default:
		log.Infof("Unhandled action from listener: %v", msg.Action)
		resourceAction = apolloTypes.Unknown
	}

	var containers []*apolloTypes.Container
	container, err := dl.getContainer(id)
	if err != nil {
		log.Infof("Failed trying to get resource (actor=%v,id=%v)", actor, id)
		if container, exists := dl.resourceCache[id]; exists {
			containers = append(containers, container)
		}
	} else {
		containers = append(containers, container)
	}

	event := apolloTypes.Event{
		Containers: containers,
		Action:     resourceAction,
	}
	dl.eventsChan <- event
}

// KillResource kills a particular resource
func (dl *Listener) KillResource(resourceType string, i interface{}) error {
	return errors.New("KillResource is not implemented")
}

func init() {
	listeners.Registry["swarm"] = func() (listenerTypes.Listener, error) {
		d, err := New()
		return d, err
	}
}
