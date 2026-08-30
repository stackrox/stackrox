package broker

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

type querySignal struct {
	requestTime time.Time
	arrived     concurrency.Signal
	resp        *central.LightspeedQueryResponse
}

// Broker coordinates and matches Lightspeed query requests to responses.
type Broker struct {
	active map[string]*querySignal
	mu     sync.Mutex
}

// New returns a new Broker instance.
func New() *Broker {
	return &Broker{
		active: make(map[string]*querySignal),
	}
}

// NotifyResponseReceived matches the ID of Sensor's response to the request and signals the waiter.
func (b *Broker) NotifyResponseReceived(resp *central.LightspeedQueryResponse) {
	if resp == nil {
		log.Warnf("Received empty Lightspeed query response, skipping")
		return
	}

	concurrency.WithLock(&b.mu, func() {
		reqID := resp.GetId()
		sig, ok := b.active[reqID]
		if !ok {
			log.Warnf("Received response to an unknown Lightspeed query request ID %s", reqID)
			return
		}

		elapsed := time.Since(sig.requestTime).Milliseconds()
		log.Debugf("Received answer for Lightspeed query request ID %s (time elapsed %dms)", reqID, elapsed)

		sig.resp = resp
		delete(b.active, reqID)
		sig.arrived.Signal()
	})
}

// SendAndWaitForSummary sends a Lightspeed query to Sensor and waits for the response.
// Returns the summary string or an error if the query failed or timed out.
func (b *Broker) SendAndWaitForSummary(ctx context.Context, conn connection.SensorConnection, query, contextJSON string, timeout time.Duration) (string, error) {
	var id string
	var sig *querySignal

	concurrency.WithLock(&b.mu, func() {
		id = uuid.NewV4().String()
		sig = &querySignal{
			requestTime: time.Now(),
			arrived:     concurrency.NewSignal(),
		}
		b.active[id] = sig
	})

	log.Debugf("Sending Lightspeed query request to Sensor with requestID %s", id)

	err := conn.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_LightspeedQueryRequest{
			LightspeedQueryRequest: &central.LightspeedQueryRequest{
				Id:          id,
				Query:       query,
				ContextJson: contextJSON,
			},
		},
	})
	if err != nil {
		// Clean up the active request on send failure
		concurrency.WithLock(&b.mu, func() {
			delete(b.active, id)
		})
		return "", errors.Wrapf(err, "failed to send message to cluster %s", conn.ClusterID())
	}

	// Wait for response or timeout
	select {
	case <-sig.arrived.Done():
		if sig.resp.GetError() != "" {
			return "", errors.Errorf("Lightspeed query failed: %s", sig.resp.GetError())
		}
		return sig.resp.GetSummary(), nil
	case <-time.After(timeout):
		// Clean up on timeout
		concurrency.WithLock(&b.mu, func() {
			delete(b.active, id)
		})
		return "", errors.New("timed out waiting for Lightspeed query response")
	}
}
