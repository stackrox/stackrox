package updatecomputer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stretchr/testify/assert"
)

func TestCategorizedComputeUpdatedProcesses(t *testing.T) {

	emptyUpdate := map[indicator.ProcessListening]timestamp.MicroTS{}
	proc1 := indicator.ProcessInfo{
		ProcessName: "foo",
		ProcessArgs: "--port 80",
		ProcessExec: "/usr/bin/foo",
	}
	proc2 := indicator.ProcessInfo{
		ProcessName: "bar",
		ProcessArgs: "--port 80",
		ProcessExec: "/usr/bin/bar",
	}
	endpointWithProcess := func(proc indicator.ProcessInfo) indicator.ProcessListening {
		return indicator.ProcessListening{
			Process:       proc,
			PodID:         "pod1",
			ContainerName: "container1",
			DeploymentID:  "deploy1",
			PodUID:        "pod_1_uid",
			Namespace:     "ns",
			Protocol:      storage.L4Protocol_L4_PROTOCOL_TCP,
			Port:          80,
		}
	}

	open := timestamp.InfiniteFuture

	tests := map[string]struct {
		initialState     map[indicator.ProcessListening]timestamp.MicroTS
		update1          map[indicator.ProcessListening]timestamp.MicroTS
		update2          map[indicator.ProcessListening]timestamp.MicroTS
		update3          map[indicator.ProcessListening]timestamp.MicroTS
		expectNumUpdates int
	}{
		// Very rare scenario (but frequent in fake workloads): an attacker may open a port using a benign process (foo),
		// and then reuse the opened port for malicious purposes (bar) without closing it, we must send an update
		// about the second process to Central.
		// The consequence of this behavior is that the `indicator.ProcessListening.keyString()` and `keyHash()`
		// must store the process information. This yields a high memory cost when running fake workloads
		// (less deduper hits), but fortunately happens very rarely in production.
		"two different processes on the same open endpoint should trigger 2 updates if no close msg in between ": {
			initialState: emptyUpdate,
			update1: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): open,
			},
			update2: emptyUpdate, // not closing the endpoint
			update3: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc2): open,
			},
			expectNumUpdates: 2,
		},
		// Typical scenario: One process closes port and another one starts listening.
		"two different processes on the same open endpoint should trigger 3 updates with close msg in between ": {
			initialState: emptyUpdate,
			update1: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): open,
			},
			update2: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): timestamp.Now(), // Closing the proc1 on the endpoint
			},
			update3: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc2): open,
			},
			expectNumUpdates: 3,
		},
		// Potential scenario: Collector reports the same data twice.
		"two identical processes on the same open endpoint should be deduped and trigger 1 update": {
			initialState: emptyUpdate,
			update1: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): open,
			},
			update2: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): open,
			},
			update3:          emptyUpdate,
			expectNumUpdates: 1,
		},
		"open->close->open for exactly the same process should trigger 3 updates": {
			initialState: emptyUpdate,
			update1: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): open,
			},
			update2: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): timestamp.Now(), // Closing the proc1 on the endpoint
			},
			update3: map[indicator.ProcessListening]timestamp.MicroTS{
				endpointWithProcess(proc1): open, // Reopening proc1 on the endpoint
			},
			expectNumUpdates: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l := NewCategorized()
			// To initialize the state of Categorized, we must trigger a single computation and call `OnSuccessfulSend`.
			_ = l.ComputeUpdatedProcesses(tc.initialState)
			l.OnSuccessfulSend(nil, nil, tc.initialState)

			_ = l.ComputeUpdatedProcesses(tc.update1)
			// Not calling `OnSuccessfulSend` to accumulate the updates
			_ = l.ComputeUpdatedProcesses(tc.update2)
			got := l.ComputeUpdatedProcesses(tc.update3)
			l.OnSuccessfulSend(nil, nil, tc.update3)

			assert.Len(t, got, tc.expectNumUpdates)

			// Empty update to ensure that any caches for offline mode are cleared
			u := l.ComputeUpdatedProcesses(emptyUpdate)
			l.OnSuccessfulSend(nil, nil, emptyUpdate)
			assert.Len(t, u, 0)
		})
	}
}
