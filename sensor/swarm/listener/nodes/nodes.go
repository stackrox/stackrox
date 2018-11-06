package nodes

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Handler implements the ResourceHandler interface
type Handler struct {
	client  *dockerClient.Client
	eventsC chan<- *v1.SensorEvent
}

// SendExistingResources sends the current node count.
func (s *Handler) SendExistingResources() {
	existingNodes, err := s.getExistingNodes()
	if err != nil {
		log.Errorf("unable to get existing nodes: %v", err)
		return
	}

	for _, n := range existingNodes {
		s.eventsC <- &v1.SensorEvent{
			Id:     n.GetId(),
			Action: v1.ResourceAction_UPDATE_RESOURCE,
			Resource: &v1.SensorEvent_Node{
				Node: n,
			},
		}
	}
}

// NewHandler instantiates the Handler for network events
func NewHandler(client *dockerClient.Client, eventsC chan<- *v1.SensorEvent) *Handler {
	return &Handler{
		client:  client,
		eventsC: eventsC,
	}
}

func (s *Handler) getExistingNodes() ([]*v1.Node, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	swarmNodes, err := s.client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	nodes := make([]*v1.Node, len(swarmNodes))
	for i, swarmNode := range swarmNodes {
		nodes[i] = &v1.Node{
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

	var resourceAction v1.ResourceAction

	switch msg.Action {
	case "create":
		resourceAction = v1.ResourceAction_CREATE_RESOURCE
	case "update":
		resourceAction = v1.ResourceAction_UPDATE_RESOURCE
	case "remove":
		resourceAction = v1.ResourceAction_REMOVE_RESOURCE
	default:
		log.Warnf("unknown action for node: %s", msg.Action)
		return
	}

	node := &v1.Node{
		Id:   msg.Actor.ID,
		Name: msg.Actor.Attributes["name"],
	}

	event := &v1.SensorEvent{
		Id:     node.GetId(),
		Action: resourceAction,
		Resource: &v1.SensorEvent_Node{
			Node: node,
		},
	}
	s.eventsC <- event
}
