package enhancement

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

// The Broker coordinates and matches deployment enhancement requests to responses
type Broker struct {
	waiters map[string]func(msg *central.DeploymentEnhancementResponse)
	lock    sync.Mutex
}

// NewBroker returns a new broker
func NewBroker() *Broker {
	return &Broker{}
}

// NotifyDeploymentReceived matches the ID of Sensors response to the request and calls the right callback for it
func (b *Broker) NotifyDeploymentReceived(msg *central.DeploymentEnhancementResponse) {
	if msg == nil || msg.GetMsg() == nil {
		log.Warnf("Received empty message, skipping enhancement notify")
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()
	if callbackFn, ok := b.waiters[msg.GetMsg().GetId()]; ok {
		log.Debugf("Received answer for Deployment enrichment requestID %s", msg.GetMsg().GetId())
		callbackFn(msg)
	}
}

// SendAndWaitForEnhancedDeployments sends a list of deployments to Sensor for additional data. Blocks while waiting.
func (b *Broker) SendAndWaitForEnhancedDeployments(ctx context.Context, conn connection.SensorConnection, deployments []*storage.Deployment, timeout time.Duration) ([]*storage.Deployment, error) {
	b.lock.Lock()
	id := uuid.NewV4().String()
	c := make(chan *central.DeploymentEnhancementResponse)
	b.waiters[id] = func(msg *central.DeploymentEnhancementResponse) {
		c <- msg
	}
	b.lock.Unlock()

	log.Debugf("Sending Deployment Augmentation request to Sensor with requestID %s", id)

	err := sendDeployments(ctx, conn, deployments, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send message to cluster %s", conn.ClusterID())
	}

	enhanced, err := b.waitAndProcessResponse(c, timeout)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process response for message %s", id)
	}

	b.lock.Lock()
	if _, ok := b.waiters[id]; ok {
		log.Debugf("Deregistering callback for id %s", id)
		delete(b.waiters, id)
	}
	b.lock.Unlock()

	return enhanced, nil
}

func (b *Broker) waitAndProcessResponse(c chan *central.DeploymentEnhancementResponse, timeout time.Duration) ([]*storage.Deployment, error) {
	var deployments []*storage.Deployment
	select {
	case m, ok := <-c:
		if !ok {
			return nil, errors.New("enhancement channel closed unexpectedly")
		}
		if deployments = m.GetMsg().GetDeployments(); deployments == nil {
			return nil, errors.New("enhanced deployments empty")
		}
		return deployments, nil
	case <-time.After(timeout):
		return nil, errors.New("timed out waiting for augmented deployment")
	}
}

func sendDeployments(ctx context.Context, conn connection.SensorConnection, deployments []*storage.Deployment, id string) error {
	return conn.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_DeploymentEnhancementRequest{
			DeploymentEnhancementRequest: &central.DeploymentEnhancementRequest{
				Msg: &central.DeploymentEnhancementMessage{
					Id:          id,
					Deployments: deployments,
				},
			},
		},
	})
}
