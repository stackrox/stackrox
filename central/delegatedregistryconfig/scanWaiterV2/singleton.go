package scanwaiterv2

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/waiter"
)

var (
	once    sync.Once
	manager waiter.Manager[*storage.ImageV2]
)

func initialize() {
	manager = waiter.NewManager[*storage.ImageV2]()
	manager.Start(context.Background())
}

// Singleton creates a single instance of a scan waiter manager.
func Singleton() waiter.Manager[*storage.ImageV2] {
	if !features.FlattenImageData.Enabled() {
		return nil
	}

	once.Do(initialize)
	return manager
}
