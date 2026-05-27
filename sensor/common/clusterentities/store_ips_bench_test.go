package clusterentities

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/set"
)

func buildIPsStoreShared(numDeployments int, withHistory bool) (*podIPsStore, net.IPAddress) {
	var memSize uint16
	if withHistory {
		memSize = 5
	}
	store := newPodIPsStoreWithMemory(memSize)

	sharedIP := net.ParseIP("10.0.0.1")
	for i := range numDeployments {
		deploymentID := fmt.Sprintf("deploy-%d", i)
		deplSet := store.ipMap[sharedIP]
		deplSet.Add(deploymentID)
		store.ipMap[sharedIP] = deplSet
		store.reverseIPMap[deploymentID] = set.NewFrozenSet(sharedIP)
	}

	if withHistory {
		histIP := net.ParseIP("10.1.0.1")
		store.historicalIPs[histIP] = make(map[string]*entityStatus)
		for i := range numDeployments {
			store.historicalIPs[histIP][fmt.Sprintf("hist-deploy-%d", i)] = newHistoricalEntity(memSize)
		}
	}

	return store, sharedIP
}

func BenchmarkLookupByNetAddr(b *testing.B) {
	deploymentCounts := []int{1, 4, 10, 50, 100}

	for _, n := range deploymentCounts {
		store, ip := buildIPsStoreShared(n, false)
		b.Run(fmt.Sprintf("%ddepl_current", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				store.LookupByNetAddr(ip, 8080)
			}
		})
	}

	for _, n := range deploymentCounts {
		store, _ := buildIPsStoreShared(n, true)
		histIP := net.ParseIP("10.1.0.1")
		b.Run(fmt.Sprintf("%ddepl_historical", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				store.LookupByNetAddr(histIP, 8080)
			}
		})
	}

	for _, n := range deploymentCounts {
		store, ip := buildIPsStoreShared(n, true)
		b.Run(fmt.Sprintf("%ddepl_current_with_history", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				store.LookupByNetAddr(ip, 8080)
			}
		})
	}
}
