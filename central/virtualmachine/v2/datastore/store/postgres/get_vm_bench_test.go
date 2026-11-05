//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BenchmarkGetVM measures the joined VM and latest-scan lookup.
// Only VM metadata and scans are seeded; the lookup does not read components or CVEs.
func BenchmarkGetVM(b *testing.B) {
	b.Setenv(features.VirtualMachinesEnhancedDataModel.EnvVar(), "true")
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)
	vmStore := New(testDB.DB, concurrency.NewKeyFence()).(*storeImpl)
	for _, vmCount := range []int{1000, 3000} {
		b.Run(fmt.Sprintf("VMs%d", vmCount), func(b *testing.B) {
			_, err := testDB.Exec(ctx, "TRUNCATE virtual_machine_v2 CASCADE")
			require.NoError(b, err)
			ids := make([]string, vmCount)
			clusterID := uuid.NewV4().String()
			tx, err := testDB.Begin(ctx)
			require.NoError(b, err)
			defer func() { _ = tx.Rollback(ctx) }()
			for i := range ids {
				ids[i] = uuid.NewV4().String()
				require.NoError(b, vmStore.insertVM(ctx, tx, &storage.VirtualMachineV2{
					Id: ids[i], Name: fmt.Sprintf("vm-%d", i), Namespace: "default",
					ClusterId: clusterID, ClusterName: "test-cluster", GuestOs: "rhel:9",
					State: storage.VirtualMachineV2_RUNNING, LastUpdated: timestamppb.Now(),
				}))
				require.NoError(b, vmStore.insertScan(ctx, tx, &storage.VirtualMachineScanV2{
					Id: uuid.NewV7().String(), VmV2Id: ids[i], ScanOs: "rhel:9",
					ScanTime: timestamppb.Now(), TopCvss: 9.8,
				}))
			}
			require.NoError(b, tx.Commit(ctx))
			_, err = testDB.Exec(ctx, "ANALYZE virtual_machine_v2, virtual_machine_scan_v2")
			require.NoError(b, err)
			b.Run("JoinedQuery", func(b *testing.B) {
				b.ReportAllocs()
				i := 0
				for b.Loop() {
					result, found, err := vmStore.GetWithLatestScan(ctx, ids[i%len(ids)])
					i++
					require.NoError(b, err)
					require.True(b, found)
					require.NotNil(b, result.Scan)
				}
			})
		})
	}
}
