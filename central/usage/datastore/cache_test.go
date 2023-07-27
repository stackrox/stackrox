package datastore

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func inject(c *cache) {
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
	assert.Equal(t, int32(0), bm.NumNodes)
	assert.Equal(t, int32(0), bm.NumCores)

	inject(c)

	ids.Add("test1")
	bm = c.FilterCurrent(ids) // removes test2 values, as not present in ids
	assert.Equal(t, int32(1), bm.NumNodes)
	assert.Equal(t, int32(10), bm.NumCores)
	ids.Add("test2")
	bm = c.FilterCurrent(ids)
	assert.Equal(t, int32(1), bm.NumNodes)
	assert.Equal(t, int32(10), bm.NumCores)

	inject(c)

	bm = c.FilterCurrent(ids)
	assert.Equal(t, int32(3), bm.NumNodes)
	assert.Equal(t, int32(30), bm.NumCores)
}

func TestCutMetrics(t *testing.T) {
	c := NewCache()

	inject(c)

	ids := set.NewStringSet()
	bm := c.CutMetrics(ids)
	assert.Equal(t, int32(0), bm.NumNodes)
	assert.Equal(t, int32(0), bm.NumCores)

	inject(c)

	ids.Add("test1")
	bm = c.CutMetrics(ids)
	assert.Equal(t, int32(1), bm.NumNodes)
	assert.Equal(t, int32(10), bm.NumCores)

	bm = c.CutMetrics(ids)
	assert.Equal(t, int32(0), bm.NumNodes)
	assert.Equal(t, int32(0), bm.NumCores)

	inject(c)

	ids.Add("test2")
	bm = c.CutMetrics(ids)
	assert.Equal(t, int32(3), bm.NumNodes)
	assert.Equal(t, int32(30), bm.NumCores)
}
