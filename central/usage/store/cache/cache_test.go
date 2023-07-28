package cache

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func inject(c Cache) {
	c.UpdateUsage("test1", &central.ClusterMetrics{
		NodeCount:   1,
		CpuCapacity: 10,
	})
	c.UpdateUsage("test2", &central.ClusterMetrics{
		NodeCount:   2,
		CpuCapacity: 20,
	})
}

func TestFilterCurrent(t *testing.T) {
	c := NewCache()
	inject(c)

	ids := set.NewStringSet()
	bm := c.FilterCurrent(ids)
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(c)

	ids.Add("test1")
	bm = c.FilterCurrent(ids) // removes test2 values, as not present in ids
	assert.Equal(t, int64(1), bm.NumNodes)
	assert.Equal(t, int64(10), bm.NumCpuUnits)
	ids.Add("test2")
	bm = c.FilterCurrent(ids)
	assert.Equal(t, int64(1), bm.NumNodes)
	assert.Equal(t, int64(10), bm.NumCpuUnits)

	inject(c)

	bm = c.FilterCurrent(ids)
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)
}

func TestCutMetrics(t *testing.T) {
	c := NewCache()

	inject(c)

	ids := set.NewStringSet()
	bm := c.CutMetrics(ids)
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(c)

	ids.Add("test1")
	bm = c.CutMetrics(ids)
	assert.Equal(t, int64(1), bm.NumNodes)
	assert.Equal(t, int64(10), bm.NumCpuUnits)

	bm = c.CutMetrics(ids)
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(c)

	ids.Add("test2")
	bm = c.CutMetrics(ids)
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)
}
