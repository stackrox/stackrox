package networks

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

// Handler implements the ResourceHandler interface
type Handler struct {
	client  *dockerClient.Client
	eventsC chan *central.SensorEvent
}

// SendExistingResources sends the current networks
func (s *Handler) SendExistingResources() {
	existingNetworks, err := s.getExistingNetworks()
	if err != nil {
		logger.Errorf("unable to get existing networks: %s", err)
		return
	}

	for _, d := range existingNetworks {
		s.eventsC <- d
	}
}

// NewHandler instantiates the Handler for network events
func NewHandler(client *dockerClient.Client, eventsC chan *central.SensorEvent) *Handler {
	return &Handler{
		client:  client,
		eventsC: eventsC,
	}
}

func (s *Handler) getNetworkPolicyFromNetworkID(id string) (*storage.NetworkPolicy, types.NetworkResource, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	networkResource, err := s.client.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		return nil, types.NetworkResource{}, err
	}
	return networkWrap(networkResource).asNetworkPolicy(), networkResource, nil
}

// HandleMessage takes a generic docker event and converts it to a network policy event
func (s *Handler) HandleMessage(msg events.Message) {
	actor := msg.Type
	id := msg.Actor.ID

	var resourceAction central.ResourceAction
	var np *storage.NetworkPolicy
	var originalSpec types.NetworkResource
	var err error

	switch msg.Action {
	case "create":
		resourceAction = central.ResourceAction_CREATE_RESOURCE
		if np, originalSpec, err = s.getNetworkPolicyFromNetworkID(id); err != nil {
			logger.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "update":
		resourceAction = central.ResourceAction_UPDATE_RESOURCE
		if np, originalSpec, err = s.getNetworkPolicyFromNetworkID(id); err != nil {
			logger.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "remove":
		resourceAction = central.ResourceAction_REMOVE_RESOURCE
		np = &storage.NetworkPolicy{
			Id: id,
		}
	default:
		resourceAction = central.ResourceAction_UNSET_ACTION_RESOURCE
		logger.Warnf("unknown action for network: %s", msg.Action)
		return
	}

	s.eventsC <- networkPolicyEventWrap(resourceAction, np, originalSpec)
}

func (s *Handler) getExistingNetworks() ([]*central.SensorEvent, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	filters := filters.NewArgs()
	filters.Add("scope", "swarm")
	swarmNetworks, err := s.client.NetworkList(ctx, types.NetworkListOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	var events []*central.SensorEvent
	for _, network := range swarmNetworks {
		n := networkWrap(network).asNetworkPolicy()
		events = append(events, networkPolicyEventWrap(central.ResourceAction_UPDATE_RESOURCE, n, network))
	}

	// Add a network policy for default namespace so all randomly run services will at least be grouped nicely
	defaultNetwork := types.NetworkResource{
		ID:   "default",
		Name: "default",
	}
	events = append(events,
		networkPolicyEventWrap(central.ResourceAction_UPDATE_RESOURCE, networkWrap(defaultNetwork).asNetworkPolicy(), defaultNetwork),
	)

	return events, nil
}

func networkPolicyEventWrap(action central.ResourceAction, networkPolicy *storage.NetworkPolicy, obj interface{}) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     networkPolicy.GetId(),
		Action: action,
		Resource: &central.SensorEvent_NetworkPolicy{
			NetworkPolicy: networkPolicy,
		},
	}
}
