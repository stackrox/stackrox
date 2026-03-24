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
