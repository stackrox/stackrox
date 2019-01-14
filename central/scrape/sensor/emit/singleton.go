package emit

import (
	"sync"

	"github.com/stackrox/rox/central/sensor/service/streamer"
)

var (
	once sync.Once

	emitter Emitter
)

// SingletonEmitter returns the singleton instance of Emitter.
func SingletonEmitter() Emitter {
	once.Do(func() {
		emitter = NewEmitter(streamer.ManagerSingleton())
	})
	return emitter
}
