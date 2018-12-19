package services

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
)

// Handler creates a new handler for sending resources
type Handler struct {
	client  *dockerClient.Client
	eventsC chan *central.SensorEvent
}

// SendExistingResources sends existing deployments
func (s *Handler) SendExistingResources() {
	existingDeployments, err := s.getNewExistingDeployments()
	if err != nil {
		log.Errorf("unable to get existing deployments: %s", err)
		return
	}

	for _, d := range existingDeployments {
		s.eventsC <- d
	}
}

// NewHandler instantiates a handler for docker services
func NewHandler(client *dockerClient.Client, eventsC chan *central.SensorEvent) *Handler {
	return &Handler{
		client:  client,
		eventsC: eventsC,
	}
}

// HandleMessage handles service message
func (s *Handler) HandleMessage(msg events.Message) {
	actor := msg.Type
	id := msg.Actor.ID

	var resourceAction central.ResourceAction
	var deployment *storage.Deployment
	var originalSpec swarm.Service
	var err error

	switch msg.Action {
	case "create":
		resourceAction = central.ResourceAction_CREATE_RESOURCE

		if deployment, originalSpec, err = s.getDeploymentFromServiceID(id); err != nil {
			log.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "update":
		resourceAction = central.ResourceAction_UPDATE_RESOURCE

		if deployment, originalSpec, err = s.getDeploymentFromServiceID(id); err != nil {
			log.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "remove":
		resourceAction = central.ResourceAction_REMOVE_RESOURCE

		deployment = &storage.Deployment{
			Id: id,
		}
	default:
		resourceAction = central.ResourceAction_UNSET_ACTION_RESOURCE
		log.Warnf("unknown action: %s", msg.Action)
		return
	}

	s.eventsC <- deploymentEventWrap(resourceAction, deployment, originalSpec)
}

func (s *Handler) getNewExistingDeployments() ([]*central.SensorEvent, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	swarmServices, err := s.client.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	deployments := make([]*central.SensorEvent, len(swarmServices))
	for i, service := range swarmServices {
		d := serviceWrap(service).asDeployment(s.client, true)
		deployments[i] = deploymentEventWrap(central.ResourceAction_UPDATE_RESOURCE, d, service)
	}
	return deployments, nil
}

func deploymentEventWrap(action central.ResourceAction, deployment *storage.Deployment, obj interface{}) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     deployment.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Deployment{
			Deployment: deployment,
		},
	}
}

func (s *Handler) getDeploymentFromServiceID(id string) (*storage.Deployment, swarm.Service, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	serviceInfo, _, err := s.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
	if err != nil {
		return nil, swarm.Service{}, err
	}
	return serviceWrap(serviceInfo).asDeployment(s.client, true), serviceInfo, nil
}
