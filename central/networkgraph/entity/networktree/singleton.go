package networktree

import (
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once
	f    Manager
)

// Singleton provides the instance of EntityDataStore to use.
func Singleton() Manager {
	once.Do(func() {
		f = newManager()
	})
	return f
}
