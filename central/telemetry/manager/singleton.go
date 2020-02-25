package manager

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	licenseSingletons "github.com/stackrox/rox/central/license/singleton"
	"github.com/stackrox/rox/central/telemetry/gatherers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
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
