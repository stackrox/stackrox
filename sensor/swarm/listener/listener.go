package listener

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/swarm/listener/networks"
	"github.com/stackrox/rox/sensor/swarm/listener/services"
)

var (
	log = logging.LoggerForModule()
)

// ResourceHandler defines an interface that will handle specific resources and docker events (e.g. services, networks)
type ResourceHandler interface {
	SendExistingResources()
	HandleMessage(events.Message)
}

// listener provides functionality for listening to deployment events.
type listener struct {
	*dockerClient.Client
	eventsC    chan *listeners.EventWrap
	stopSig    concurrency.Signal
	stoppedSig concurrency.Signal

	resourceHandlers map[string]ResourceHandler
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
	eventsC := make(chan *listeners.EventWrap, 10)
	return &listener{
		Client:     dockerClient,
		eventsC:    eventsC,
		stopSig:    concurrency.NewSignal(),
		stoppedSig: concurrency.NewSignal(),
		resourceHandlers: map[string]ResourceHandler{
			"service": services.NewServiceHandler(dockerClient, eventsC),
			"network": networks.NewHandler(dockerClient, eventsC),
		},
	}, nil
}

// Start starts the listener
func (dl *listener) Start() {
	events, errors, cancel := dl.eventHandler()

	// Send all existing resources
	for _, handler := range dl.resourceHandlers {
		handler.SendExistingResources()
	}

	log.Info("Swarm Listener Started")
	defer dl.stoppedSig.Signal()
	for {
		select {
		case event := <-events:
			handler, ok := dl.resourceHandlers[event.Type]
			if !ok {
				log.Warnf("Event type '%s' does not have a defined handler", event.Type)
				continue
			}
			handler.HandleMessage(event)
		case err := <-errors:
			log.Infof("Reopening stream due to error: %+v", err)
			// Provide a small amount of time for the potential issue to correct itself
			time.Sleep(1 * time.Second)
			events, errors, cancel = dl.eventHandler()
		case <-dl.stopSig.Done():
			log.Infof("Shutting down Swarm Listener")
			cancel()
			return
		}
	}
}

func (dl *listener) eventHandler() (<-chan (events.Message), <-chan error, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	filters := filters.NewArgs()
	filters.Add("scope", "swarm")
	filters.Add("type", "service")
	filters.Add("type", "network")
	events, errors := dl.Client.Events(ctx, types.EventsOptions{
		Filters: filters,
	})
	return events, errors, cancel
}

// Events is the mechanism through which the events are propagated back to the event loop
func (dl *listener) Events() <-chan *listeners.EventWrap {
	return dl.eventsC
}

func (dl *listener) Stop() {
	dl.stopSig.Signal()
	dl.stoppedSig.Wait()
}
