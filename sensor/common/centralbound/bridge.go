package centralbound

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/unimplemented"
)

const defaultBufferSize = 100

// Bridge is a SensorComponent that subscribes to the CentralBound PubSub
// topic and forwards received events to a ResponsesC channel. This allows
// components to publish Central-bound messages via PubSub instead of
// maintaining their own ResponsesC channels, enabling incremental migration.
type Bridge struct {
	unimplemented.Receiver

	responsesC chan *message.ExpiringMessage
	stopper    concurrency.Stopper
}

// NewBridge creates a bridge that subscribes to the CentralBound PubSub topic
// and exposes received messages through ResponsesC.
func NewBridge(dispatcher common.PubSubDispatcher) (*Bridge, error) {
	b := &Bridge{
		responsesC: make(chan *message.ExpiringMessage, defaultBufferSize),
		stopper:    concurrency.NewStopper(),
	}
	if err := dispatcher.RegisterConsumerToLane(
		pubsub.CentralBoundBridgeConsumer,
		pubsub.CentralBoundTopic,
		pubsub.CentralBoundLane,
		b.handleEvent,
	); err != nil {
		return nil, errors.Wrap(err, "registering central-bound bridge consumer")
	}
	return b, nil
}

func (b *Bridge) handleEvent(e pubsub.Event) error {
	evt, ok := e.(*CentralBoundEvent)
	if !ok {
		return errors.Errorf("unexpected event type: %T", e)
	}
	if evt.IsExpired() {
		return nil
	}
	select {
	case b.responsesC <- evt.Msg:
	case <-b.stopper.Flow().StopRequested():
	}
	return nil
}

// Start is a no-op; the consumer is registered at construction time.
func (b *Bridge) Start() error { return nil }

// Stop closes the ResponsesC channel so centralSenderImpl's forwardResponses
// goroutine exits cleanly.
func (b *Bridge) Stop() {
	b.stopper.Client().Stop()
	close(b.responsesC)
}

// Notify is a no-op; the bridge does not react to lifecycle events.
func (b *Bridge) Notify(_ common.SensorComponentEvent) {}

// Capabilities returns nil; the bridge has no capabilities to advertise.
func (b *Bridge) Capabilities() []centralsensor.SensorCapability { return nil }

// ResponsesC returns the channel carrying messages destined for Central.
func (b *Bridge) ResponsesC() <-chan *message.ExpiringMessage { return b.responsesC }

// Name returns the component name used in logging.
func (b *Bridge) Name() string { return "centralbound.Bridge" }
