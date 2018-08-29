package services

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/listeners"
)

// Handler creates a new handler for sending resources
type Handler struct {
	client  *dockerClient.Client
	eventsC chan *listeners.EventWrap
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

// NewServiceHandler instantiates a handler for docker services
func NewServiceHandler(client *dockerClient.Client, eventsC chan *listeners.EventWrap) *Handler {
	return &Handler{
		client:  client,
		eventsC: eventsC,
	}
}

// HandleMessage handles service message
func (s *Handler) HandleMessage(msg events.Message) {
	actor := msg.Type
	id := msg.Actor.ID

	var resourceAction v1.ResourceAction
	var deployment *v1.Deployment
	var originalSpec swarm.Service
	var err error

	switch msg.Action {
	case "create":
		resourceAction = v1.ResourceAction_CREATE_RESOURCE

		if deployment, originalSpec, err = s.getDeploymentFromServiceID(id); err != nil {
			log.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "update":
		resourceAction = v1.ResourceAction_UPDATE_RESOURCE

		if deployment, originalSpec, err = s.getDeploymentFromServiceID(id); err != nil {
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

	s.eventsC <- deploymentEventWrap(resourceAction, deployment, originalSpec)
}

func (s *Handler) getNewExistingDeployments() ([]*listeners.EventWrap, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	swarmServices, err := s.client.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	deployments := make([]*listeners.EventWrap, len(swarmServices))
	for i, service := range swarmServices {
		d := serviceWrap(service).asDeployment(s.client, true)
		deployments[i] = deploymentEventWrap(v1.ResourceAction_UPDATE_RESOURCE, d, service)
	}
	return deployments, nil
}

func deploymentEventWrap(action v1.ResourceAction, deployment *v1.Deployment, obj interface{}) *listeners.EventWrap {
	return &listeners.EventWrap{
		SensorEvent: &v1.SensorEvent{
			Id:     deployment.GetId(),
			Action: action,
			Resource: &v1.SensorEvent_Deployment{
				Deployment: deployment,
			},
		},
		OriginalSpec: obj,
	}
}

func (s *Handler) getDeploymentFromServiceID(id string) (*v1.Deployment, swarm.Service, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	serviceInfo, _, err := s.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
	if err != nil {
		return nil, swarm.Service{}, err
	}
	return serviceWrap(serviceInfo).asDeployment(s.client, true), serviceInfo, nil
}
