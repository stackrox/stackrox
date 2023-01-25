package declarativeconfig

import "github.com/stackrox/rox/pkg/sync"

var (
	once     sync.Once
	instance Manager
)

// Singleton provides the instance of Manager to use.
func Singleton() Manager {
	once.Do(func() {
		instance = New()
	})
	return instance
}
