package scanwaiter

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/waiter"
)

var (
	once    sync.Once
	manager waiter.Manager[*storage.Image]
)

func initialize() {
	manager = waiter.NewManager[*storage.Image]()
	manager.Start(context.Background())
}

// Singleton creates a single instance of a scan waiter manager.
func Singleton() waiter.Manager[*storage.Image] {
	once.Do(initialize)
	return manager
}
