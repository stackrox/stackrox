package clusterentities

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/set"
)

func doLookupEndpointOld[M Map[T], T any](ep net.NumericEndpoint, src map[net.NumericEndpoint]map[string]M) (results []LookupResult) {
	for deploymentID, targetInfoSet := range src[ep] {
		result := LookupResult{
			Entity:         networkgraph.EntityForDeployment(deploymentID),
			ContainerPorts: make([]uint16, 0),
		}
		for tgtInfo := range targetInfoSet {
			result.ContainerPorts = append(result.ContainerPorts, tgtInfo.ContainerPort)
			if tgtInfo.PortName != "" {
				result.PortNames = append(result.PortNames, tgtInfo.PortName)
			}
		}
		results = append(results, result)
	}
	return results
}

func buildLookupBenchData(numDeployments, targetsPerDeployment int) (net.NumericEndpoint, map[net.NumericEndpoint]map[string]set.Set[EndpointTargetInfo]) {
	ep := buildEndpoint("10.0.0.1", 8080)
	deploymentsMap := make(map[string]set.Set[EndpointTargetInfo], numDeployments)
	for d := range numDeployments {
		etiSet := make(set.Set[EndpointTargetInfo], targetsPerDeployment)
		for t := range targetsPerDeployment {
			etiSet.Add(EndpointTargetInfo{
				ContainerPort: uint16(8080 + t),
				PortName:      fmt.Sprintf("port-%d", t),
			})
		}
		deploymentsMap[fmt.Sprintf("depl-%d", d)] = etiSet
	}
	src := map[net.NumericEndpoint]map[string]set.Set[EndpointTargetInfo]{ep: deploymentsMap}
	return ep, src
}

func BenchmarkDoLookupEndpoint(b *testing.B) {
	deploymentCounts := []int{1, 10, 50, 100}
	targetCounts := []int{1, 4, 8}

	for _, numDeployments := range deploymentCounts {
		for _, targetsPerDeployment := range targetCounts {
			name := fmt.Sprintf("%ddepl_%dtargets", numDeployments, targetsPerDeployment)
			ep, src := buildLookupBenchData(numDeployments, targetsPerDeployment)

			b.Run("old/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					doLookupEndpointOld(ep, src)
				}
			})
			b.Run("new/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					doLookupEndpoint(ep, src)
				}
			})
		}
	}
}
