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
	requests map[string]chan<- *central.DeploymentEnhancementResponse
	lock     sync.Mutex
}

// NewBroker returns a new broker
func NewBroker() *Broker {
	return &Broker{
		requests: make(map[string]chan<- *central.DeploymentEnhancementResponse),
	}
}

// NotifyDeploymentReceived matches the ID of Sensors response to the request and notifies the waiting goroutine
func (b *Broker) NotifyDeploymentReceived(msg *central.DeploymentEnhancementResponse) {
	if msg == nil || msg.GetMsg() == nil {
		log.Warnf("Received empty message, skipping enhancement notify")
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()
	if r, ok := b.requests[msg.GetMsg().GetId()]; ok {
		// Once, to prevent writing to a closed channel if a msg dupe arrives
		log.Debugf("Received answer for Deployment enrichment requestID %v", msg.GetMsg().GetId())
		r <- msg
	}
}

// SendAndWaitForEnhancedDeployments sends a list of deployments to Sensor for additional data. Blocks while waiting.
func (b *Broker) SendAndWaitForEnhancedDeployments(ctx context.Context, conn connection.SensorConnection, deployments []*storage.Deployment, timeout time.Duration) ([]*storage.Deployment, error) {
	b.lock.Lock()
	ch := make(chan *central.DeploymentEnhancementResponse)
	id := uuid.NewV4().String()
	b.requests[id] = ch
	b.lock.Unlock()

	log.Debugf("Sending Deployment Augmentation request to Sensor with requestID %s", id)

	err := conn.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_DeploymentEnhancementRequest{
			DeploymentEnhancementRequest: &central.DeploymentEnhancementRequest{
				Msg: &central.DeploymentEnhancementMessage{
					Id:          id,
					Deployments: deployments,
				},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send message to cluster %s", conn.ClusterID())
	}

	select {
	case m, ok := <-ch:
		if !ok {
			return nil, errors.New("augmented channel closed unexpectedly")
		}
		if deployments := m.GetMsg().GetDeployments(); deployments == nil {
			return nil, errors.New("augmented deployments empty")
		}
		b.lock.Lock()
		if r, ok := b.requests[id]; ok {
			close(r)
			delete(b.requests, id)
		}
		defer b.lock.Unlock()
		return deployments, nil
	case <-time.After(timeout):
		return nil, errors.New("timed out waiting for augmented deployment")
	}

}
