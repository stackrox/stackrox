package clusterentities

import (
	"fmt"
	"testing"
)

func applyBenchGenerateEntityData(numEndpoints, targetsPerEndpoint int) *EntityData {
	data := &EntityData{}
	// Duplicate endpoints
	// Duplicate endpoints, different ports, same port names - makes no sense, but that is the duplication scenario
	endpoint1 := buildEndpoint("10.0.0.1", 8080)
	endpoint2 := buildEndpoint("10.0.0.1", 8081)
	data.AddEndpoint(endpoint1, EndpointTargetInfo{
		ContainerPort: 8080,
		PortName:      "http",
	})
	data.AddEndpoint(endpoint2, EndpointTargetInfo{
		ContainerPort: 8080,
		PortName:      "http",
	})
	// Same ports different port IPs
	endpoint3 := buildEndpoint("10.0.0.1", 8082)
	endpoint4 := buildEndpoint("10.0.0.2", 8082)

	data.AddEndpoint(endpoint3, EndpointTargetInfo{
		ContainerPort: 8082,
		PortName:      "http",
	})
	data.AddEndpoint(endpoint4, EndpointTargetInfo{
		ContainerPort: 8082,
		PortName:      "http",
	})

	endpoint := buildEndpoint("10.0.0.3", 8080)
	data.AddEndpoint(endpoint, EndpointTargetInfo{
		ContainerPort: 8080,
		PortName:      "http",
	})
	data.AddEndpoint(endpoint, EndpointTargetInfo{
		ContainerPort: 8080,
		PortName:      "http",
	})

	// Random endpoints
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

func BenchmarkEndpointsUnchangedNoLock(b *testing.B) {
	for _, tc := range []struct {
		numEndpoints       int
		targetsPerEndpoint int
	}{
		{numEndpoints: 10, targetsPerEndpoint: 10},
		{numEndpoints: 50, targetsPerEndpoint: 10},
		{numEndpoints: 200, targetsPerEndpoint: 1},
		{numEndpoints: 200, targetsPerEndpoint: 2},
		{numEndpoints: 200, targetsPerEndpoint: 4},
		{numEndpoints: 200, targetsPerEndpoint: 8},
		{numEndpoints: 200, targetsPerEndpoint: 16},
		{numEndpoints: 200, targetsPerEndpoint: 32},
	} {
		name := fmt.Sprintf("ep%d_tgt%d", tc.numEndpoints, tc.targetsPerEndpoint)
		b.Run(name, func(b *testing.B) {
			b.Run("baseline", func(b *testing.B) {
				store := newEndpointsStoreWithMemory(5)
				data := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
				store.applyNoLock(map[string]*EntityData{"depl-bench": data}, false)

				for b.Loop() {
					if !store.endpointsUnchangedNoLockBaseline("depl-bench", data.endpoints) {
						b.Fatal("expected endpoints to be unchanged")
					}
				}
			})
			b.Run("optimized", func(b *testing.B) {
				store := newEndpointsStoreWithMemory(5)
				data := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
				store.applyNoLock(map[string]*EntityData{"depl-bench": data}, false)

				for b.Loop() {
					if !store.endpointsUnchangedNoLock("depl-bench", data.endpoints) {
						b.Fatal("expected endpoints to be unchanged")
					}
				}
			})
			b.Run("hybrid", func(b *testing.B) {
				store := newEndpointsStoreWithMemory(5)
				data := applyBenchGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
				store.applyNoLock(map[string]*EntityData{"depl-bench": data}, false)

				for b.Loop() {
					if !store.endpointsUnchangedNoLockHybrid("depl-bench", data.endpoints) {
						b.Fatal("expected endpoints to be unchanged")
					}
				}
			})
		})
	}
}
