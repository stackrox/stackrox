package datastore

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
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

// vmV2Walker is the subset of the VM V2 datastore this gatherer needs.
type vmV2Walker interface {
	Walk(ctx context.Context, fn func(vm *storage.VirtualMachineV2) error) error
}

// scanV2Counter is the subset of the scan V2 datastore this gatherer needs.
type scanV2Counter interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// GatherV2 returns a gatherer that queries the V2 VM and scan datastores.
func GatherV2(vmV2DS vmV2Walker, scanV2DS scanV2Counter) phonehome.GatherFunc {
	return gatherV2WithTime(vmV2DS, scanV2DS, time.Now)
}

func gatherV2WithTime(vmV2DS vmV2Walker, scanV2DS scanV2Counter, nowFunc func() time.Time) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		if !features.VirtualMachines.Enabled() || !features.VirtualMachinesEnhancedDataModel.Enabled() {
			return map[string]any{}, nil
		}

		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.VirtualMachine),
			),
		)

		clusterIDsWithRunningVMs := set.NewStringSet()
		totalVMs := 0

		err := vmV2DS.Walk(ctx, func(vm *storage.VirtualMachineV2) error {
			totalVMs++
			if vm.GetState() == storage.VirtualMachineV2_RUNNING {
				clusterID := vm.GetClusterId()
				if clusterID == "" {
					log.Debugf("Virtual machine V2 %s has empty cluster_id", vm.GetId())
				} else {
					clusterIDsWithRunningVMs.Add(clusterID)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		now := nowFunc()
		cutoff := now.Add(-activeVMAgentMaxAgeLimitTelemetry)
		cutoffTS := protocompat.ConvertTimeToTimestampOrNil(&cutoff)
		q := search.NewQueryBuilder().
			AddStrings(search.VirtualMachineScanTime, ">"+protocompat.ConvertTimestampToString(cutoffTS, time.RFC3339Nano)).
			ProtoQuery()

		vmsWithActiveAgents, err := scanV2DS.Count(ctx, q)
		if err != nil {
			return nil, err
		}

		props := map[string]any{
			metricClustersWithVMs:     clusterIDsWithRunningVMs.Cardinality(),
			metricTotalVMs:            totalVMs,
			metricVMsWithActiveAgents: vmsWithActiveAgents,
		}
		log.Debugf("Virtual machines V2 telemetry update: %v", props)
		return props, nil
	}
}
