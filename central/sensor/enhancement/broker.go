package enhancement

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

type enhancementSignal struct {
	requestTime time.Time
	msgArrived  concurrency.Signal
	msg         *central.DeploymentEnhancementResponse
}

// The Broker coordinates and matches deployment enhancement requests to responses
type Broker struct {
	activeRequests map[string]*enhancementSignal
	lock           sync.Mutex
}

// NewBroker returns a new broker
func NewBroker() *Broker {
	return &Broker{
		activeRequests: make(map[string]*enhancementSignal),
	}
}

// NotifyDeploymentReceived matches the ID of Sensors response to the request and calls the right callback for it
func (b *Broker) NotifyDeploymentReceived(msg *central.DeploymentEnhancementResponse) {
	if msg == nil || msg.GetMsg() == nil {
		log.Warnf("Received empty message, skipping enhancement notify")
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()
	reqID := msg.GetMsg().GetId()
	sig, ok := b.activeRequests[reqID]
	if !ok {
		log.Warnf("Received response to an unknown Deployment Enrichment Request ID %s", reqID)
		return
	}
	elapsed := time.Since(sig.requestTime).Milliseconds()
	log.Debugf("Received answer for Deployment Enrichment Request ID %s (time elapsed %dms)", reqID, elapsed)
	sig.msg = msg
	metrics.ObserveDeploymentEnhancementTime(float64(elapsed))
	delete(b.activeRequests, reqID)
	sig.msgArrived.Signal()
}

// SendAndWaitForEnhancedDeployments sends a list of deployments to Sensor for additional data. Blocks while waiting.
func (b *Broker) SendAndWaitForEnhancedDeployments(ctx context.Context, conn connection.SensorConnection, deployments []*storage.Deployment, timeout time.Duration) ([]*storage.Deployment, error) {
	var id string
	var s enhancementSignal

	concurrency.WithLock(&b.lock, func() {
		id = uuid.NewV4().String()
		s = enhancementSignal{
			requestTime: time.Now(),
			msgArrived:  concurrency.NewSignal(),
		}
		b.activeRequests[id] = &s
	})

	log.Debugf("Sending Deployment Augmentation request to Sensor with requestID %s", id)

	err := sendDeployments(ctx, conn, deployments, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send message to cluster %s", conn.ClusterID())
	}

	enhanced, err := b.waitAndProcessResponse(&s, timeout)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process response for message %s", id)
	}

	return enhanced, nil
}

func (b *Broker) waitAndProcessResponse(s *enhancementSignal, timeout time.Duration) ([]*storage.Deployment, error) {
	var deployments []*storage.Deployment
	select {
	case <-s.msgArrived.Done():
		if deployments = s.msg.GetMsg().GetDeployments(); deployments == nil {
			return nil, errors.New("enhanced deployments empty")
		}
		return deployments, nil
	case <-time.After(timeout):
		return nil, errors.New("timed out waiting for enhanced deployment")
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
