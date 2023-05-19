package scanwaiter

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/waiter"
)

var (
	once sync.Once

	m waiter.Manager[*storage.Image]
)

func initialize() {
	m = waiter.NewManager[*storage.Image]()
	m.Start(context.Background())
}

// Singleton creates a single instance of a scan waiter manager
func Singleton() waiter.Manager[*storage.Image] {
	once.Do(initialize)
	return m
}
