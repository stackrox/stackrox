package clusterentities

import (
	"fmt"
	"testing"
)

// applyBenchGenerateEntityData builds synthetic endpoint data for benchmarking.
// All endpoints share the same external port (8080) with varying IPs, each
// carrying targetsPerEndpoint distinct target infos with sequential container
// ports and unique port names.
func applyBenchGenerateEntityData(numEndpoints, targetsPerEndpoint int) *EntityData {
	data := &EntityData{}
	for endpointIdx := range numEndpoints {
		endpoint := buildEndpoint(fmt.Sprintf("10.%d.%d.%d", (endpointIdx/65536)%256, (endpointIdx/256)%256, endpointIdx%256), 8080)
		for targetIdx := range targetsPerEndpoint {
			data.AddEndpoint(endpoint, EndpointTargetInfo{
				ContainerPort: uint16(8080 + targetIdx),
				PortName:      fmt.Sprintf("port-%d", targetIdx),
			})
		}
	}
	return data
}

func BenchmarkApplyUnchanged(b *testing.B) {
	for _, tc := range []struct {
		numEndpoints       int
		targetsPerEndpoint int
	}{
		{numEndpoints: 10, targetsPerEndpoint: 2},
		{numEndpoints: 50, targetsPerEndpoint: 4},
		{numEndpoints: 200, targetsPerEndpoint: 4},
	} {
		name := fmt.Sprintf("ep%d_tgt%d", tc.numEndpoints, tc.targetsPerEndpoint)
		b.Run(name, func(b *testing.B) {
			store := newEndpointsStoreWithMemory(5)
			data := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
			updates := map[string]*EntityData{"depl-bench": data}
			store.Apply(updates, false)

			for b.Loop() {
				store.Apply(updates, false)
			}
		})
	}
}

func BenchmarkApplyChanged(b *testing.B) {
	for _, tc := range []struct {
		numEndpoints       int
		targetsPerEndpoint int
	}{
		{numEndpoints: 10, targetsPerEndpoint: 2},
		{numEndpoints: 50, targetsPerEndpoint: 4},
		{numEndpoints: 200, targetsPerEndpoint: 4},
	} {
		name := fmt.Sprintf("ep%d_tgt%d", tc.numEndpoints, tc.targetsPerEndpoint)
		b.Run(name, func(b *testing.B) {
			store := newEndpointsStoreWithMemory(5)
			dataA := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
			dataB := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint+1)
			updatesA := map[string]*EntityData{"depl-bench": dataA}
			updatesB := map[string]*EntityData{"depl-bench": dataB}
			store.Apply(updatesA, false)

			for i := 0; b.Loop(); i++ {
				if i%2 == 0 {
					store.Apply(updatesB, false)
				} else {
					store.Apply(updatesA, false)
				}
			}
		})
	}
}

// BenchmarkApplyPartialChange measures the scenario from ROX-34642: a deployment
// with many endpoints where only one endpoint changes per update. The baseline
// purge-all/reinsert-all approach moves all N endpoints to history and back even
// when only 1 changed.
func BenchmarkApplyPartialChange(b *testing.B) {
	for _, tc := range []struct {
		numEndpoints       int
		targetsPerEndpoint int
	}{
		{numEndpoints: 50, targetsPerEndpoint: 2},
		{numEndpoints: 200, targetsPerEndpoint: 2},
		{numEndpoints: 200, targetsPerEndpoint: 4},
		{numEndpoints: 500, targetsPerEndpoint: 2},
	} {
		name := fmt.Sprintf("ep%d_tgt%d", tc.numEndpoints, tc.targetsPerEndpoint)
		b.Run(name, func(b *testing.B) {
			store := newEndpointsStoreWithMemory(5)
			dataA := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
			updatesA := map[string]*EntityData{"depl-bench": dataA}
			store.Apply(updatesA, false)

			// Build a variant with one endpoint's target info changed.
			dataB := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
			for ep, tis := range dataB.endpoints {
				if len(tis) > 0 {
					tis[0] = EndpointTargetInfo{ContainerPort: 9999, PortName: "changed"}
					dataB.endpoints[ep] = tis
					break
				}
			}
			updatesB := map[string]*EntityData{"depl-bench": dataB}

			for i := 0; b.Loop(); i++ {
				if i%2 == 0 {
					store.Apply(updatesB, false)
				} else {
					store.Apply(updatesA, false)
				}
			}
		})
	}
}

// BenchmarkEndpointsUnchangedNoLock measures the unchanged-endpoints fast path
// across all implementation variants.
//
// Realistic scenarios (based on how endpoints.go builds EntityData):
//   - Each endpoint normally carries 1 target info (one service port mapping).
//   - Endpoint count is driven by service type: ~10 for ClusterIP (pod IPs),
//     ~50-200 for NodePort (one per node IP), 200-500 for large NodePort clusters.
//
// Stress-test scenarios use elevated targets-per-endpoint (8-32) to exercise
// algorithmic scaling; these do not represent typical Kubernetes configurations.
func BenchmarkEndpointsUnchangedNoLock(b *testing.B) {
	for _, tc := range []struct {
		name               string
		numEndpoints       int
		targetsPerEndpoint int
	}{
		// Realistic: 1-2 targets per endpoint, growing endpoint count.
		// ClusterIP → pod IPs only; NodePort → one endpoint per node IP.
		{name: "clusterip_small", numEndpoints: 10, targetsPerEndpoint: 2},
		{name: "clusterip_medium", numEndpoints: 50, targetsPerEndpoint: 2},
		{name: "nodeport_medium", numEndpoints: 100, targetsPerEndpoint: 2},
		{name: "nodeport_large", numEndpoints: 200, targetsPerEndpoint: 2},
		{name: "nodeport_xlarge", numEndpoints: 500, targetsPerEndpoint: 2},

		// Stress test: targets per endpoint beyond realistic values.
		{name: "stress_tgt4", numEndpoints: 500, targetsPerEndpoint: 4},
		{name: "stress_tgt8", numEndpoints: 500, targetsPerEndpoint: 8},
		{name: "stress_tgt16", numEndpoints: 500, targetsPerEndpoint: 16},
		{name: "stress_tgt32", numEndpoints: 500, targetsPerEndpoint: 32},
	} {
		b.Run(tc.name, func(b *testing.B) {
			store := newEndpointsStoreWithMemory(5)
			data := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
			store.applyNoLock(map[string]*EntityData{"depl-bench": data}, false)

			for b.Loop() {
				store.endpointsUnchangedNoLock("depl-bench", data.endpoints)
			}
		})
	}
}
