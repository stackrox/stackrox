package enhancement

import "github.com/stackrox/rox/pkg/sync"

var (
	brokerInstance     *Broker
	brokerInstanceInit sync.Once
)

// BrokerSingleton returns the singleton instance for the broker that manages sensor deployment enhancement requests
func BrokerSingleton() *Broker {
	brokerInstanceInit.Do(func() {
		brokerInstance = NewBroker()
	})
	return brokerInstance
}
