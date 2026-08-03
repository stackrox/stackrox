package scannerclient

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

var (
	_ common.Notifiable = (*resetNotifiable)(nil)

	notifiable     *resetNotifiable
	notifiableOnce sync.Once
)

// ResetNotifiable returns a notifiable that resets
// the gRPC client singleton when notified.
func ResetNotifiable(pubSubDispatcher common.PubSubDispatcher) common.Notifiable {
	notifiableOnce.Do(func() {
		notifiable = &resetNotifiable{pubSubDispatcher: pubSubDispatcher}
		if notifiable.pubSubEnabled() {
			if err := pubSubDispatcher.RegisterConsumerToLane(
				pubsub.ScannerClientResetSensorOnlineConsumer,
				pubsub.SensorOnlineTopic,
				pubsub.SensorOnlineLane,
				notifiable.handleSensorOnlineEvent,
			); err != nil {
				log.Errorf("failed to register scanner client reset sensor online consumer: %v", err)
			}
		}
	})

	return notifiable
}

// resetNotifiable will reset the scanner client singleton when notified.
// This allows the scanner client to be recreated on next retrieval, allowing
// for re-evaluation, for example, of central capabilities.
type resetNotifiable struct {
	common.Notifiable

	pubSubDispatcher common.PubSubDispatcher
}

func (r *resetNotifiable) pubSubEnabled() bool {
	return features.SensorInternalPubSub.Enabled() && r.pubSubDispatcher != nil
}

func (r *resetNotifiable) handleSensorOnlineEvent(event pubsub.Event) error {
	if _, ok := event.(*events.SensorOnlineEvent); !ok {
		return errors.Errorf("unexpected event type: %T", event)
	}
	resetGRPCClient()
	log.Debug("Reset scanner client")
	return nil
}

func (r *resetNotifiable) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	if r.pubSubEnabled() {
		// Online transitions are handled by the SensorOnlineEvent PubSub
		// subscription registered in ResetNotifiable().
		return
	}
	switch e {
	case common.SensorComponentEventCentralReachable:
		resetGRPCClient()
		log.Debug("Reset scanner client")
	}
}
