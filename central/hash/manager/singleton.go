package manager

import (
	"context"

	"github.com/stackrox/rox/central/hash/datastore"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	manager Manager
	once    sync.Once
)

// Singleton returns the hash flush manager
func Singleton() Manager {
	once.Do(func() {
		manager = NewManager(datastore.Singleton())

		go manager.Start(sac.WithAllAccess(context.Background()))
	})
	return manager
}
