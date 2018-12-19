package nodes

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Handler implements the ResourceHandler interface
type Handler struct {
	client  *dockerClient.Client
	eventsC chan<- *central.SensorEvent
}

// SendExistingResources sends the current node count.
func (s *Handler) SendExistingResources() {
	existingNodes, err := s.getExistingNodes()
	if err != nil {
		log.Errorf("unable to get existing nodes: %v", err)
		return
	}

	for _, n := range existingNodes {
		s.eventsC <- &central.SensorEvent{
			Id:     n.GetId(),
			Action: central.ResourceAction_UPDATE_RESOURCE,
			Resource: &central.SensorEvent_Node{
				Node: n,
			},
		}
	}
}

// NewHandler instantiates the Handler for network events
func NewHandler(client *dockerClient.Client, eventsC chan<- *central.SensorEvent) *Handler {
	return &Handler{
		client:  client,
		eventsC: eventsC,
	}
}

func (s *Handler) getExistingNodes() ([]*storage.Node, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	swarmNodes, err := s.client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	nodes := make([]*storage.Node, len(swarmNodes))
	for i, swarmNode := range swarmNodes {
		nodes[i] = &storage.Node{
			Id:   swarmNode.ID,
			Name: swarmNode.Description.Hostname,
		}
	}
	return nodes, nil
}

// HandleMessage takes a generic docker event and converts it to a network policy event
func (s *Handler) HandleMessage(msg events.Message) {
	if msg.Type != "node" {
		return
	}

	var resourceAction central.ResourceAction

	switch msg.Action {
	case "create":
		resourceAction = central.ResourceAction_CREATE_RESOURCE
	case "update":
		resourceAction = central.ResourceAction_UPDATE_RESOURCE
	case "remove":
		resourceAction = central.ResourceAction_REMOVE_RESOURCE
	default:
		log.Warnf("unknown action for node: %s", msg.Action)
		return
	}

	node := &storage.Node{
		Id:   msg.Actor.ID,
		Name: msg.Actor.Attributes["name"],
	}

	event := &central.SensorEvent{
		Id:     node.GetId(),
		Action: resourceAction,
		Resource: &central.SensorEvent_Node{
			Node: node,
		},
	}
	s.eventsC <- event
}
