package manager

import (
	"context"

	"github.com/stackrox/stackrox/central/globaldb"
	licenseSingletons "github.com/stackrox/stackrox/central/license/singleton"
	"github.com/stackrox/stackrox/central/telemetry/gatherers"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	instance     Manager
	instanceInit sync.Once

	allAccessContext = sac.WithAllAccess(context.Background())
)

// Singleton returns the license manager singleton instance.
func Singleton() Manager {
	instanceInit.Do(func() {
		var err error
		instance, err = NewManager(allAccessContext, globaldb.GetGlobalDB(), gatherers.Singleton(), licenseSingletons.ManagerSingleton())
		if err != nil {
			panic(err)
		}
	})
	return instance
}
