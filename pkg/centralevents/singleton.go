package centralevents

import (
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	stream Stream
)

// Singleton returns an instance of the Central events stream.
func Singleton() Stream {
	if !features.CentralEvents.Enabled() {
		return nil
	}
	once.Do(func() {
		stream = newStream()
	})
	return stream
}
