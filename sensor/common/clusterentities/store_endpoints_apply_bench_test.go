package clusterentities

import (
	"fmt"
	"testing"
)

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

			b.ResetTimer()
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

			b.ResetTimer()
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
