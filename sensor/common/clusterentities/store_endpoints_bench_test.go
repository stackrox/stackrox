package clusterentities

import (
	"fmt"
	"testing"
)

// BenchmarkApply measures the performance of the Apply function with realistic workloads
func BenchmarkApply(b *testing.B) {
	benchmarks := []struct {
		name            string
		numDeployments  int
		endpointsPerDep int
		targetsPerEp    int
		incremental     bool
	}{
		{"Small_Incremental", 10, 5, 2, true},
		{"Medium_Incremental", 50, 10, 3, true},
		{"Large_Incremental", 100, 20, 4, true},
		{"Small_Full", 10, 5, 2, false},
		{"Medium_Full", 50, 10, 3, false},
		{"Large_Full", 100, 20, 4, false},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create realistic test data
			updates := generateTestUpdates(bm.numDeployments, bm.endpointsPerDep, bm.targetsPerEp)

			for b.Loop() {
				store := newEndpointsStoreWithMemory(100)
				store.Apply(updates, bm.incremental)
			}
		})
	}
}

// BenchmarkApplySingleNoLock measures the performance of applySingleNoLock directly
func BenchmarkApplySingleNoLock(b *testing.B) {
	benchmarks := []struct {
		name           string
		numEndpoints   int
		targetsPerEp   int
		deploymentName string
	}{
		{"SingleDeployment_5Endpoints", 5, 2, "deployment-1"},
		{"SingleDeployment_20Endpoints", 20, 3, "deployment-1"},
		{"SingleDeployment_50Endpoints", 50, 4, "deployment-1"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			data := generateEntityData(bm.numEndpoints, bm.targetsPerEp)
			store := newEndpointsStoreWithMemory(100)

			for b.Loop() {
				store.mutex.Lock()
				store.applySingleNoLock(bm.deploymentName, *data)
				store.mutex.Unlock()
			}
		})
	}
}

// BenchmarkApplyRepeated measures the performance of repeated Apply calls on the same store
// This simulates real-world usage where the store is updated over time
func BenchmarkApplyRepeated(b *testing.B) {
	store := newEndpointsStoreWithMemory(100)
	updates := generateTestUpdates(50, 10, 3)

	for b.Loop() {
		store.Apply(updates, true)
	}
}

// Helper functions to generate test data

// generateTestUpdates creates a map of deployment updates with realistic data
func generateTestUpdates(numDeployments, endpointsPerDep, targetsPerEp int) map[string]*EntityData {
	updates := make(map[string]*EntityData, numDeployments)
	for d := range numDeployments {
		deploymentID := fmt.Sprintf("deployment-%d", d)
		updates[deploymentID] = generateEntityData(endpointsPerDep, targetsPerEp)
	}
	return updates
}

// generateEntityData creates EntityData with the specified number of endpoints and targets
func generateEntityData(numEndpoints, targetsPerEp int) *EntityData {
	ed := &EntityData{}
	for e := range numEndpoints {
		// Generate realistic IP addresses (10.x.y.z range)
		ip := fmt.Sprintf("10.%d.%d.%d", (e/256)/256, (e/256)%256, e%256)
		port := uint16(8080 + (e % 20)) // Vary ports realistically

		ep := buildEndpoint(ip, port)

		// Add multiple targets per endpoint
		for t := range targetsPerEp {
			targetPort := port + uint16(t)
			portName := fmt.Sprintf("port-%d", targetPort)
			ed.AddEndpoint(ep, EndpointTargetInfo{
				ContainerPort: targetPort,
				PortName:      portName,
			})
		}
	}
	return ed
}
