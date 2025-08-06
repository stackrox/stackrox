package updatecomputer

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

type dataSet struct {
	data       []map[indicator.NetworkConn]timestamp.MicroTS
	currentIdx int
}

func (d *dataSet) Generate(numSets, totalConnections, beingClosed, newOpenConnections int) {
	d.currentIdx = 0
	d.data = make([]map[indicator.NetworkConn]timestamp.MicroTS, numSets)
	current, previous := generateConnectionMaps(totalConnections, beingClosed, newOpenConnections)
	d.data[0], d.data[1] = previous, current
	for i := 2; i < numSets; i++ {
		_, d.data[i] = generateConnectionMaps(totalConnections, beingClosed, newOpenConnections)
	}
}

func (d *dataSet) Next() map[indicator.NetworkConn]timestamp.MicroTS {
	if d.currentIdx >= len(d.data) {
		d.currentIdx = 0
	}
	return d.data[d.currentIdx]
}

func (d *dataSet) ResetIdx() {
	d.currentIdx = 0
}

var ds map[int]*dataSet
var sizes = []int{1000, 10_000, 100_000}

func init() {
	ds = make(map[int]*dataSet, len(sizes))
	for _, size := range sizes {
		ds[size] = &dataSet{}
		beingClosed := int(float64(size) * 0.05)
		newConns := int(float64(size) * 0.05)
		ds[size].Generate(200, size, beingClosed, newConns)
	}
}

// BenchmarkUpdateComputerMemoryUsage compares memory usage between implementations
func BenchmarkUpdateComputerMemoryUsage(b *testing.B) {
	for _, size := range sizes {
		ds[size].ResetIdx()
		compl := NewLegacy()
		compl.UpdateState(ds[size].Next(), nil, nil)
		compc := NewCategorized()
		compc.UpdateState(ds[size].Next(), nil, nil)

		b.Run(fmt.Sprintf("Legacy_%d_connections", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = compl.ComputeUpdatedConns(ds[size].Next())
			}
		})

		b.Run(fmt.Sprintf("Categorized_%d_connections", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = compc.ComputeUpdatedConns(ds[size].Next())
			}
		})
	}

}

//// BenchmarkUpdateComputerPerformance compares CPU performance between implementations
//func BenchmarkUpdateComputerPerformance(b *testing.B) {
//	scenarios := map[string]struct {
//		connectionCount int
//		numClosing      int
//		numNewOpen      int
//	}{
//		"SmallDataset_HighChurn": {1000, 500, 0},
//		"SmallDataset_LowChurn":  {1000, 50, 0},
//		"LargeDataset_HighChurn": {100_000, 50_000, 0},
//		"LargeDataset_LowChurn":  {100_000, 5_000, 0},
//	}
//
//}

// Helper functions for generating test data

func generateConnectionMaps(totalConnections, beingClosed, newOpenConnections int) (map[indicator.NetworkConn]timestamp.MicroTS, map[indicator.NetworkConn]timestamp.MicroTS) {
	current := make(map[indicator.NetworkConn]timestamp.MicroTS, totalConnections+newOpenConnections)
	previous := make(map[indicator.NetworkConn]timestamp.MicroTS, totalConnections+newOpenConnections)
	if beingClosed >= totalConnections {
		panic("kept open count must be less than total connections")
	}

	now := timestamp.Now()
	for i := 0; i < totalConnections; i++ {
		conn := generateConnection(i)
		previous[*conn] = timestamp.InfiniteFuture
		if i < beingClosed {
			current[*conn] = now
		} else {
			current[*conn] = timestamp.InfiniteFuture
		}
	}
	for i := 0; i < newOpenConnections; i++ {
		conn := generateConnection(totalConnections + i)
		current[*conn] = timestamp.InfiniteFuture
	}

	return current, previous
}

func generateConnection(id int) *indicator.NetworkConn {
	srcEntity := networkgraph.Entity{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		ID:   fmt.Sprintf("deployment-src-%d", id%100), // Reuse some entities
	}
	dstEntity := networkgraph.Entity{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		ID:   fmt.Sprintf("deployment-dst-%d", (id+50)%100),
	}

	return &indicator.NetworkConn{
		SrcEntity: srcEntity,
		DstEntity: dstEntity,
		DstPort:   uint16(80 + (id % 1000)), // Vary ports
		Protocol:  storage.L4Protocol_L4_PROTOCOL_TCP,
	}
}
