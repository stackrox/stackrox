package docker

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/listeners"
	listenerTypes "bitbucket.org/stack-rox/apollo/apollo/listeners/types"
	apolloTypes "bitbucket.org/stack-rox/apollo/apollo/types"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/pkg/errors"
)

var (
	log = logging.New("listener/docker")
)

const (
	// DefaultDockerAPIVersion is the Docker API version we will use in the
	// absence of an exact version we can detect at runtime.
	// This should be the API version for the minimum Docker version we support.
	// For Docker version to API version table, see:
	//   https://docs.docker.com/engine/reference/api/docker_remote_api/
	defaultDockerAPIVersion = 1.22

	dockerHangTimeout = 30 * time.Second
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
	dockerClient, err := newDockerClient()
	if err != nil {
		return nil, err
	}
	if err := negotiateClientVersionToLatest(dockerClient, defaultDockerAPIVersion); err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), dockerHangTimeout)
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
	ctx, cancel := context.WithTimeout(context.Background(), dockerHangTimeout)
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

	log.Infof("Docker Msg: %+v", msg)
	var resourceAction apolloTypes.ResourceAction
	switch msg.Action {
	case "create":
		resourceAction = apolloTypes.Create
	case "remove":
		resourceAction = apolloTypes.Remove
	case "update":
		resourceAction = apolloTypes.Update
	default:
		resourceAction = apolloTypes.Unknown
	}

	container, err := dl.getContainer(id)
	if err != nil {
		log.Infof("Failed trying to get resource (actor=%v,id=%v)", actor, id)
		container = dl.resourceCache[id]
	}

	event := apolloTypes.Event{
		Containers: []*apolloTypes.Container{container},
		Action:     resourceAction,
	}
	log.Infof("%+v", event)
	dl.eventsChan <- event
}

// newDockerClient returns a new docker client or an error if there was issues generating it
func newDockerClient() (*dockerClient.Client, error) {
	client, err := dockerClient.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Unable to create docker client: %+v", err)
	}
	return client, nil
}

func getDockerVersion(v string) (float64, error) {
	version, err := strconv.ParseFloat(v, 64)
	return version, err
}

func dockerVersionString(v float64) string {
	return fmt.Sprintf("%0.2f", v)
}

// negotiateClientVersionToLatest negotiates the golang API version with the Docker server
func negotiateClientVersionToLatest(client dockerClient.APIClient, dockerAPIVersion float64) error {
	// update client version to lowest supported version
	client.UpdateClientVersion(dockerVersionString(defaultDockerAPIVersion))
	versionStruct, err := client.ServerVersion(context.Background())
	if err != nil {
		return fmt.Errorf("unable to get docker server version: %+v", err)
	}
	var minClientVersion float64
	if versionStruct.MinAPIVersion == "" { // Backwards compatibility
		minClientVersion, err = getDockerVersion(versionStruct.APIVersion)
		if err != nil {
			return fmt.Errorf("unable to parse docker server api version: %+v", err)
		}
	} else {
		minClientVersion, err = getDockerVersion(versionStruct.MinAPIVersion)
		if err != nil {
			return fmt.Errorf("unable to parse docker server min api version: %+v", err)
		}
	}
	versionToNegotiate := dockerAPIVersion
	if dockerAPIVersion < minClientVersion {
		versionToNegotiate = minClientVersion
	}
	log.Infof("Negotiating Docker API version to %v", versionToNegotiate)
	client.UpdateClientVersion(dockerVersionString(versionToNegotiate))
	return nil
}

// KillResource kills a particular resource
func (dl *Listener) KillResource(resourceType string, i interface{}) error {
	return errors.New("KillResource is not implemented")
}

func init() {
	listeners.Registry["docker"] = func() (listenerTypes.Listener, error) {
		d, err := New()
		return d, err
	}
}
