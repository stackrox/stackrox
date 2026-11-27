//go:build sql_integration && workload_experiments

package benchmark_test

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
)

// TestWorkloadPreparedExecutions measures complete two-query reads and records which
// plan PostgreSQL used for the event statement on each execution.
func TestWorkloadPreparedExecutions(t *testing.T) {
	fixture := newWorkloadBenchmark(t, workloadBenchmarkSize{100000, 200000, 100000, 100000}, "auto")
	ctx := sac.WithAllAccess(context.Background())
	recorder := &workloadQueryRecorder{DB: fixture.db}
	fixture.db = recorder
	for _, read := range []string{"count", "page"} {
		t.Run(read, func(t *testing.T) {
			expected := fixture.expectedCount
			if read == "page" {
				expected = min(expected, 50)
			}
			for i := 1; i <= 6; i++ {
				start := time.Now()
				actual, err := fixture.run(ctx, read, true)
				elapsed := time.Since(start)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
				var generic, custom int64
				require.NoError(t, recorder.DB.QueryRow(ctx,
					"SELECT generic_plans, custom_plans FROM pg_prepared_statements WHERE statement = $1", recorder.sql).
					Scan(&generic, &custom))
				t.Logf("execution=%d elapsed=%s generic_plans=%d custom_plans=%d", i, elapsed, generic, custom)
			}
		})
	}
}
