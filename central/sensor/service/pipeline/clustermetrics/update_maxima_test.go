package clustermetrics

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func inject() {
	updateMaxima("test1", &central.ClusterMetrics{
		NodeCount:   1,
		CpuCapacity: 10,
	})
	updateMaxima("test2", &central.ClusterMetrics{
		NodeCount:   2,
		CpuCapacity: 20,
	})
}

func TestFilterCurrent(t *testing.T) {
	inject()

	ids := set.NewStringSet()
	bm := FilterCurrent(ids)
	assert.Equal(t, int64(0), bm.TotalNodes)
	assert.Equal(t, int64(0), bm.TotalCores)

	inject()

	ids.Add("test1")
	bm = FilterCurrent(ids) // removes test2 values, as not present in ids
	assert.Equal(t, int64(1), bm.TotalNodes)
	assert.Equal(t, int64(10), bm.TotalCores)
	ids.Add("test2")
	bm = FilterCurrent(ids)
	assert.Equal(t, int64(1), bm.TotalNodes)
	assert.Equal(t, int64(10), bm.TotalCores)

	inject()

	bm = FilterCurrent(ids)
	assert.Equal(t, int64(3), bm.TotalNodes)
	assert.Equal(t, int64(30), bm.TotalCores)
}

func TestCutMetrics(t *testing.T) {
	inject()

	ids := set.NewStringSet()
	bm := CutMetrics(ids)
	assert.Equal(t, int64(0), bm.TotalNodes)
	assert.Equal(t, int64(0), bm.TotalCores)

	inject()

	ids.Add("test1")
	bm = CutMetrics(ids)
	assert.Equal(t, int64(1), bm.TotalNodes)
	assert.Equal(t, int64(10), bm.TotalCores)

	bm = CutMetrics(ids)
	assert.Equal(t, int64(0), bm.TotalNodes)
	assert.Equal(t, int64(0), bm.TotalCores)

	inject()

	ids.Add("test2")
	bm = CutMetrics(ids)
	assert.Equal(t, int64(3), bm.TotalNodes)
	assert.Equal(t, int64(30), bm.TotalCores)
}
