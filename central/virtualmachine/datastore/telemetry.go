package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

const (
	// activeVMAgentMaxAgeLimitTelemetry defines how recent a scan must be to consider
	// the VM agent as active. This threshold is included in the telemetry metric name.
	activeVMAgentMaxAgeLimitTelemetry = 24 * time.Hour

	// Metric name constants to ensure consistency between implementation and tests
	metricClustersWithVMs     = "Total Secured Clusters With Virtual Machines"
	metricTotalVMs            = "Total Virtual Machines"
	metricVMsWithActiveAgents = "Total Virtual Machines With Active Agents (Last 24h)"
)

var (
	log = logging.LoggerForModule()
)

// Gather returns a function that collects telemetry about virtual machines.
// It tracks three metrics:
// 1. Number of distinct secured clusters with at least one running VM
// 2. Total number of virtual machines
// 3. Number of VMs with active agents (received IndexReport within last 24 hours)
//
// When the ROX_VIRTUAL_MACHINES feature flag is disabled, this function returns
// an empty map without performing any database queries, ensuring no performance impact.
func Gather(ds DataStore) phonehome.GatherFunc {
	return gatherWithTime(ds, time.Now)
}

// gatherWithTime allows injecting a time function for deterministic testing.
func gatherWithTime(ds DataStore, nowFunc func() time.Time) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		// Early return if virtual machines feature is disabled - zero performance impact
		if !features.VirtualMachines.Enabled() {
			return map[string]any{}, nil
		}

		// Use elevated permissions for telemetry gathering
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.VirtualMachine),
			),
		)

		// Process VMs one-by-one without loading all into memory
		clusterIDsWithRunningVMs := set.NewStringSet()
		totalVMs := 0
		vmsWithActiveAgents := 0
		now := nowFunc()

		err := ds.Walk(ctx, func(vm *storage.VirtualMachine) error {
			totalVMs++

			// Count VMs with active agents (scan received within threshold)
			if scan := vm.GetScan(); scan != nil {
				scanTime, err := protocompat.ConvertTimestampToTimeOrError(scan.GetScanTime())
				if err != nil {
					log.Debugf("Virtual machine %s has invalid scan_time: %v", vm.GetId(), err)
				} else if now.Sub(scanTime) <= activeVMAgentMaxAgeLimitTelemetry {
					vmsWithActiveAgents++
				}
			}

			// Count clusters with RUNNING virtual machines
			if vm.GetState() == storage.VirtualMachine_RUNNING {
				clusterID := vm.GetClusterId()
				if clusterID == "" {
					// Log empty cluster IDs at debug level for troubleshooting
					log.Debugf("Virtual machine %s has empty cluster_id", vm.GetId())
				} else {
					clusterIDsWithRunningVMs.Add(clusterID)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		props := map[string]any{
			metricClustersWithVMs:     clusterIDsWithRunningVMs.Cardinality(),
			metricTotalVMs:            totalVMs,
			metricVMsWithActiveAgents: vmsWithActiveAgents,
		}
		log.Debugf("Virtual machines telemetry update: %v", props)
		return props, nil
	}
}
