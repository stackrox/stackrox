package broker

import "github.com/stackrox/rox/pkg/sync"

var (
	instance     *Broker
	instanceInit sync.Once
)

// Singleton returns the singleton instance of the Lightspeed broker.
func Singleton() *Broker {
	instanceInit.Do(func() {
		instance = New()
	})
	return instance
}
