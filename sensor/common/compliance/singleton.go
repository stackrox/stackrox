package compliance

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

// Singleton implements a singleton for the client streaming gRPC service between collector and sensor
func Singleton() Service {
	once.Do(func() {
		as = NewService()
	})
	return as
}
