package secrets

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
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

// SendExistingResources sends the current secrets
func (s *Handler) SendExistingResources() {
	existingSecrets, err := s.getExistingSecrets()
	if err != nil {
		logger.Errorf("unable to get existing secrets: %s", err)
		return
	}

	for _, d := range existingSecrets {
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

func (s *Handler) getSecretFromSecretID(id string) (*storage.Secret, swarm.Secret, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	secretResource, _, err := s.client.SecretInspectWithRaw(ctx, id)
	if err != nil {
		return nil, swarm.Secret{}, err
	}
	return secretWrap(secretResource).asSecret(), secretResource, nil
}

// HandleMessage takes a generic docker event and converts it to a network policy event
func (s *Handler) HandleMessage(msg events.Message) {
	actor := msg.Type
	id := msg.Actor.ID

	var resourceAction central.ResourceAction
	var secret *storage.Secret
	var originalSpec swarm.Secret
	var err error

	switch msg.Action {
	case "create":
		resourceAction = central.ResourceAction_CREATE_RESOURCE
		if secret, originalSpec, err = s.getSecretFromSecretID(id); err != nil {
			logger.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "update":
		resourceAction = central.ResourceAction_UPDATE_RESOURCE
		if secret, originalSpec, err = s.getSecretFromSecretID(id); err != nil {
			logger.Errorf("unable to get deployment (actor=%v,id=%v): %s", actor, id, err)
			return
		}
	case "remove":
		resourceAction = central.ResourceAction_REMOVE_RESOURCE
		secret = &storage.Secret{
			Id: id,
		}
	default:
		resourceAction = central.ResourceAction_UNSET_ACTION_RESOURCE
		logger.Warnf("unknown action for network: %s", msg.Action)
		return
	}

	s.eventsC <- secretEventWrap(resourceAction, secret, originalSpec)
}

func (s *Handler) getExistingSecrets() ([]*central.SensorEvent, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	swarmSecrets, err := s.client.SecretList(ctx, types.SecretListOptions{})
	if err != nil {
		return nil, err
	}

	var events []*central.SensorEvent
	for _, secret := range swarmSecrets {
		s := secretWrap(secret).asSecret()
		events = append(events, secretEventWrap(central.ResourceAction_UPDATE_RESOURCE, s, secret))
	}
	return events, nil
}

func secretEventWrap(action central.ResourceAction, secret *storage.Secret, obj interface{}) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     secret.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Secret{
			Secret: secret,
		},
	}
}
