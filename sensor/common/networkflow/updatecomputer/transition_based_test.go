package updatecomputer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	"github.com/stretchr/testify/assert"
)

func TestTransitionBasedComputeUpdatedProcesses(t *testing.T) {

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
	processListening := func(proc indicator.ProcessInfo) *indicator.ProcessListening {
		return &indicator.ProcessListening{
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

	type update struct {
		p  *indicator.ProcessListening
		ts timestamp.MicroTS
	}

	tests := map[string]struct {
		updates          []update
		expectNumUpdates int
	}{
		// Very rare scenario (but frequent in fake workloads): an attacker may open a port using a benign process (foo),
		// and then reuse the opened port for malicious purposes (bar) without closing it, we must send an update
		// about the second process to Central.
		// The consequence of this behavior is that the `indicator.ProcessListening.keyString()` and `keyHash()`
		// must store the process information. This yields a high memory cost when running fake workloads
		// (less deduper hits), but fortunately happens very rarely in production.
		"two different processes on the same open endpoint should trigger 2 updates if no close msg in between ": {
			updates: []update{
				{p: processListening(proc1), ts: open},
				{}, // not closing the endpoint
				{p: processListening(proc2), ts: open},
			},
			expectNumUpdates: 2,
		},
		// Typical scenario: One process closes port and another one starts listening.
		"two different processes on the same open endpoint should trigger 3 updates with close msg in between ": {
			updates: []update{
				{p: processListening(proc1), ts: open},
				{p: processListening(proc1), ts: timestamp.Now()}, // Closing the proc1 on the endpoint
				{p: processListening(proc2), ts: open},
			},
			expectNumUpdates: 3,
		},
		// Potential scenario: Collector reports the same data twice.
		"two identical processes on the same open endpoint should be deduped and trigger 1 update": {
			updates: []update{
				{p: processListening(proc1), ts: open},
				{p: processListening(proc1), ts: open},
				{}, // empty update
			},
			expectNumUpdates: 1,
		},
		"open->close->open for exactly the same process should trigger 3 updates": {
			updates: []update{
				{p: processListening(proc1), ts: open},
				{p: processListening(proc1), ts: timestamp.Now()}, // Closing the proc1 on the endpoint
				{p: processListening(proc1), ts: open},            // Reopening proc1 on the endpoint
			},
			expectNumUpdates: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l := NewTransitionBased()
			// Exemplary endpoint used across this test
			ep := indicator.ContainerEndpoint{
				Entity:   networkgraph.EntityForDeployment("dummy-id"),
				Port:     80,
				Protocol: net.TCP.ToProtobuf(),
			}

			// Initialize state with empty maps
			initialEpProc := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{}
			l.ComputeUpdatedEndpointsAndProcesses(initialEpProc)
			l.OnSuccessfulSendEndpoints(initialEpProc)
			l.OnSuccessfulSendProcesses(initialEpProc)

			var gotProc []*storage.ProcessListeningOnPortFromSensor

			// Apply all updates in online mode
			for _, upd := range tc.updates {
				procEpMap := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{}
				if upd.p != nil {
					procEpMap[ep] = &indicator.ProcessListeningWithClose{
						ProcessListening: upd.p,
						LastSeen:         upd.ts,
					}
				}
				_, updatedProc := l.ComputeUpdatedEndpointsAndProcesses(procEpMap)
				gotProc = append(gotProc, updatedProc...)
				l.OnSuccessfulSendEndpoints(procEpMap)
				l.OnSuccessfulSendProcesses(procEpMap)
			}

			assert.Len(t, gotProc, tc.expectNumUpdates)

			// Empty update to ensure that any caches for offline mode are cleared
			empty := map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithClose{}
			uEp, uProc := l.ComputeUpdatedEndpointsAndProcesses(empty)
			l.OnSuccessfulSendEndpoints(empty)
			l.OnSuccessfulSendProcesses(empty)
			assert.Len(t, uEp, 0)
			assert.Len(t, uProc, 0)
		})
	}
}
