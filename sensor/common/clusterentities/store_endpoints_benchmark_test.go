package clusterentities

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/set"
)

func benchmarkSeedEndpointsStore(numEndpoints int) (*endpointsStore, string, net.NumericEndpoint) {
	const deploymentID = "depl-bench"
	store := newEndpointsStoreWithMemory(5)
	epSet := set.NewSet[net.NumericEndpoint]()

	var firstEndpoint net.NumericEndpoint
	for i := range numEndpoints {
		ep := buildEndpoint(fmt.Sprintf("10.%d.%d.%d", (i/65536)%256, (i/256)%256, i%256), 80)
		if i == 0 {
			firstEndpoint = ep
		}
		etiSet := set.NewSet[EndpointTargetInfo]()
		etiSet.Add(EndpointTargetInfo{
			ContainerPort: 80,
			PortName:      "http",
		})
		store.endpointMap[ep] = map[string]set.Set[EndpointTargetInfo]{deploymentID: etiSet}
		epSet.Add(ep)
	}
	store.reverseEndpointMap[deploymentID] = epSet

	return store, deploymentID, firstEndpoint
}

func BenchmarkEndpointsStoreAddToHistory(b *testing.B) {
	for _, tc := range []struct {
		name         string
		numEndpoints int
	}{
		{name: "endpoints_100", numEndpoints: 100},
		{name: "endpoints_1000", numEndpoints: 1000},
		{name: "endpoints_5000", numEndpoints: 5000},
	} {
		b.Run(tc.name, func(b *testing.B) {
			store, deploymentID, endpoint := benchmarkSeedEndpointsStore(tc.numEndpoints)
			b.ResetTimer()
			for b.Loop() {
				// Keep the current-map shape constant and measure addToHistory only.
				store.historicalEndpoints = make(map[net.NumericEndpoint]map[string]map[EndpointTargetInfo]*entityStatus)
				store.reverseHistoricalEndpoints = make(map[string]map[net.NumericEndpoint]*entityStatus)
				store.addToHistory(deploymentID, endpoint)
			}
		})
	}
}

func TestApplyNoLock_AllocationsForUnchangedDeployment(t *testing.T) {
	store := newEndpointsStoreWithMemory(5)
	data := benchmarkGenerateEntityData(50, 4)
	updates := map[string]*EntityData{
		"depl-bench": data,
	}
	store.Apply(updates, false)

	allocs := testing.AllocsPerRun(100, func() {
		store.Apply(updates, false)
	})

	const (
		targetAllocs = 5.0  // desired steady-state budget for unchanged Apply
		maxAllocs    = 10.0 // upper bound to catch regressions while tolerating minor runtime variations
	)

	if allocs > maxAllocs {
		t.Fatalf("expected unchanged Apply to stay within allocation budget (target <= %.0f, max tolerated <= %.0f), got %.0f allocations",
			targetAllocs, maxAllocs, allocs)
	}

	if allocs > targetAllocs {
		t.Logf("unchanged Apply allocations above target budget: got %.0f allocations (target <= %.0f, max tolerated <= %.0f)",
			allocs, targetAllocs, maxAllocs)
	}
}

func benchmarkGenerateEntityData(numEndpoints, targetsPerEndpoint int) *EntityData {
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

// legacy is the version used in 4.10.0 and earlier (before backporting the fix).
/*
Running tool: /usr/local/go/bin/go test -test.fullpath=true -benchmem -run=^$ -bench ^BenchmarkEndpointsStoreAddToHistory$ github.com/stackrox/rox/sensor/common/clusterentities -count=1

goos: darwin
goarch: arm64
pkg: github.com/stackrox/rox/sensor/common/clusterentities
cpu: Apple M3 Pro
BenchmarkEndpointsStoreAddToHistory/endpoints_100-12         	 1656640	       695.7 ns/op	    2100 B/op	      12 allocs/op
BenchmarkEndpointsStoreAddToHistory/legacy_endpoints_100-12  	  129315	      9212 ns/op	   20226 B/op	     120 allocs/op
BenchmarkEndpointsStoreAddToHistory/endpoints_1000-12        	 1849548	       650.1 ns/op	    2100 B/op	      12 allocs/op
BenchmarkEndpointsStoreAddToHistory/legacy_endpoints_1000-12 	   10000	    115167 ns/op	  302780 B/op	    1031 allocs/op
BenchmarkEndpointsStoreAddToHistory/endpoints_5000-12        	 1718923	       703.6 ns/op	    2100 B/op	      12 allocs/op
BenchmarkEndpointsStoreAddToHistory/legacy_endpoints_5000-12 	    2221	    535328 ns/op	 1195998 B/op	    5057 allocs/op
PASS
ok  	github.com/stackrox/rox/sensor/common/clusterentities	7.895s
*/

func BenchmarkApplySingleNoLock_AllocationsForNewDeployment(b *testing.B) {
	for _, tc := range []struct {
		name               string
		numEndpoints       int
		targetsPerEndpoint int
	}{
		{name: "50x4", numEndpoints: 50, targetsPerEndpoint: 4},
		{name: "500x4", numEndpoints: 500, targetsPerEndpoint: 4},
		{name: "5000x4", numEndpoints: 5000, targetsPerEndpoint: 4},
	} {
		b.Run(tc.name, func(b *testing.B) {
			data := benchmarkGenerateEntityData(tc.numEndpoints, tc.targetsPerEndpoint)
			b.ReportAllocs()
			for b.Loop() {
				store := newEndpointsStoreWithMemory(5)
				store.applySingleNoLock("depl-bench", *data)
			}
		})
	}
}

func benchmarkGenerateEntityData(numEndpoints, targetsPerEndpoint int) *EntityData {
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
