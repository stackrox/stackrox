package networks

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

// Handler implements the ResourceHandler interface
type Handler struct {
	client  *dockerClient.Client
	eventsC chan *listeners.EventWrap
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
func NewHandler(client *dockerClient.Client, eventsC chan *listeners.EventWrap) *Handler {
	return &Handler{
		client:  client,
		eventsC: eventsC,
	}
}

func (s *Handler) getNetworkPolicyFromNetworkID(id string) (*v1.NetworkPolicy, types.NetworkResource, error) {
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

	var resourceAction v1.ResourceAction
	var np *v1.NetworkPolicy
	var originalSpec types.NetworkResource
	var err error

	switch msg.Action {
	case "create":
		resourceAction = v1.ResourceAction_CREATE_RESOURCE
		if np, originalSpec, err = s.getNetworkPolicyFromNetworkID(id); err != nil {
			logger.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "update":
		resourceAction = v1.ResourceAction_UPDATE_RESOURCE
		if np, originalSpec, err = s.getNetworkPolicyFromNetworkID(id); err != nil {
			logger.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "remove":
		resourceAction = v1.ResourceAction_REMOVE_RESOURCE
		np = &v1.NetworkPolicy{
			Id: id,
		}
	default:
		resourceAction = v1.ResourceAction_UNSET_ACTION_RESOURCE
		logger.Warnf("unknown action for network: %s", msg.Action)
		return
	}

	s.eventsC <- networkPolicyEventWrap(resourceAction, np, originalSpec)
}

func (s *Handler) getExistingNetworks() ([]*listeners.EventWrap, error) {
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

	var events []*listeners.EventWrap
	for _, network := range swarmNetworks {
		n := networkWrap(network).asNetworkPolicy()
		events = append(events, networkPolicyEventWrap(v1.ResourceAction_PREEXISTING_RESOURCE, n, network))
	}

	// Add a network policy for default namespace so all randomly run services will at least be grouped nicely
	defaultNetwork := types.NetworkResource{
		ID:   "default",
		Name: "default",
	}
	events = append(events,
		networkPolicyEventWrap(v1.ResourceAction_PREEXISTING_RESOURCE, networkWrap(defaultNetwork).asNetworkPolicy(), defaultNetwork),
	)

	return events, nil
}

func networkPolicyEventWrap(action v1.ResourceAction, networkPolicy *v1.NetworkPolicy, obj interface{}) *listeners.EventWrap {
	return &listeners.EventWrap{
		SensorEvent: &v1.SensorEvent{
			Id:     networkPolicy.GetId(),
			Action: action,
			Resource: &v1.SensorEvent_NetworkPolicy{
				NetworkPolicy: networkPolicy,
			},
		},
		OriginalSpec: obj,
	}
}
