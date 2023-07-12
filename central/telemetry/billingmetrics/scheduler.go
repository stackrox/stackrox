package billingmetrics

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/telemetry/gatherers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
)

const period = 1 * time.Hour

var (
	sig concurrency.Signal
)

func gather() {
	debugCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	adminCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	if data := gatherers.Singleton().Gather(debugCtx, true, false); data != nil {
		updateMaxima(adminCtx, data.Clusters)
	}
}

// Schedule starts a periodic data gathering from the secured clusters, updating
// the maximus store with the total numbers of secured nodes and millicores.
func Schedule() {
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		gather()
		for {
			select {
			case <-ticker.C:
				gather()
			case <-sig.Done():
				sig.Reset()
				return
			}
		}
	}()
}

// Stop stops the scheduled timer
func Stop() {
	sig.Signal()
}
