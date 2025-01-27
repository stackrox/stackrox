package droprate

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	rateMessage "github.com/stackrox/rox/sensor/common/internalmessage/rate"
)

var (
	log = logging.LoggerForModule()

	textFromKind = map[string]string{
		internalmessage.SensorMessageDropRateHigh:   "message drop rate is high",
		internalmessage.SensorMessageDropRateNormal: "message drop rate is normal",
	}
)

type RateManager interface {
	Record()
}

// NewRateManager creates a new RateManager
func NewRateManager(name string, size int, itemDropC chan struct{}, stopC concurrency.ReadOnlyErrorSignal, rateTime time.Duration, rateLimit int, pubSub *internalmessage.MessageSubscriber) RateManager {
	return rate.NewManager(
		itemDropC,
		stopC,
		rateTime,
		rateLimit,
		getCallback(internalmessage.SensorMessageDropRateHigh, name, size, pubSub),
		getCallback(internalmessage.SensorMessageDropRateNormal, name, size, pubSub),
	)
}

func getTextFromKind(kind string) string {
	if text, ok := textFromKind[kind]; ok {
		return text
	}
	return "unknown kind"
}

func getCallback(kind string, name string, size int, pubSub *internalmessage.MessageSubscriber) func(int) {
	return func(numDropped int) {
		if err := pubSub.Publish(&internalmessage.SensorInternalMessage{
			Kind: kind,
			Text: getTextFromKind(kind),
			Payload: &rateMessage.Payload{
				QueueName:  name,
				QueueSize:  size,
				NumDropped: numDropped,
			},
		}); err != nil {
			log.Errorf("unable to publish message: %v", err)
		}
	}
}
