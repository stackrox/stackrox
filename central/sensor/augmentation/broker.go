package augmentation

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

type Broker struct {
	requests map[string]chan<- *central.DeploymentEnhancementResponse
	lock     sync.Mutex
}

func NewBroker() *Broker {
	return &Broker{}
}

// NotifyDeploymentReceived .
func (b *Broker) NotifyDeploymentReceived(msg *central.DeploymentEnhancementResponse) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if r, ok := b.requests[msg.GetDeployment().GetId()]; ok {
		select {
		case r <- msg:
			// Write message to the right channel and close it
			close(r)
			break
		default:
			// In case Sensor sends multiple messages, this could deadlock.
			// Discard message to avoid locking central.
		}

	}
}

// SendAndWaitForAugmentedDeployments .
func (b Broker) SendAndWaitForAugmentedDeployments(ctx context.Context, conn connection.SensorConnection, deployments []*storage.Deployment, timeout time.Duration) ([]*storage.Deployment, error) {
	b.lock.Lock()
	ch := make(chan *central.DeploymentEnhancementResponse, 1)
	id := uuid.NewV4().String()
	b.requests[id] = ch
	b.lock.Unlock()

	err := conn.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_DeploymentEnhancementRequest{
			DeploymentEnhancementRequest: &central.DeploymentEnhancementRequest{
				Deployment: &central.DeploymentEnhancementMessage{
					Id:         id,
					Deployment: deployments,
				},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send message to cluster %s", deployments[0].GetClusterId()) // TODO: This seems risky
	}

	select {
	case m, ok := <-ch:
		if !ok {
			return nil, errors.New("augmented channel closed unexpectedly")
		}
		if deployments := m.GetDeployment().GetDeployment(); deployments == nil {
			return nil, errors.New("augmented deployments empty") // TODO: Is this really an error?
		}
		return deployments, nil
	case <-time.After(timeout):
		return nil, errors.New("timed out waiting for augmented deployment")
	}
}
