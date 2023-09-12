package cache

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func inject(c Cache) {
	c.UpdateUsage("test1", &storage.SecuredUnits{
		NumNodes:    1,
		NumCpuUnits: 10,
	})
	c.UpdateUsage("test2", &storage.SecuredUnits{
		NumNodes:    2,
		NumCpuUnits: 20,
	})
}

func TestCleanupCurrent(t *testing.T) {
	c := NewCache()

	inject(c)

	ids := set.NewStringSet()
	c.Cleanup(ids)
	bm := c.GetCurrent()
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(c)

	ids.Add("test1")
	c.Cleanup(ids)
	bm = c.GetCurrent() // removes test2 values, as not present in ids
	assert.Equal(t, int64(1), bm.NumNodes)
	assert.Equal(t, int64(10), bm.NumCpuUnits)
	ids.Add("test2")
	c.Cleanup(ids)
	bm = c.GetCurrent()
	assert.Equal(t, int64(1), bm.NumNodes)
	assert.Equal(t, int64(10), bm.NumCpuUnits)

	inject(c)

	c.Cleanup(ids)
	bm = c.GetCurrent()
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)
}

func TestAggregateAndReset(t *testing.T) {
	c := NewCache()

	inject(c)
	bm := c.AggregateAndReset()
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)

	ids := set.NewStringSet()
	c.Cleanup(ids)
	bm = c.AggregateAndReset()
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(c)

	ids.Add("test1")
	c.Cleanup(ids)
	bm = c.AggregateAndReset()
	assert.Equal(t, int64(1), bm.NumNodes)
	assert.Equal(t, int64(10), bm.NumCpuUnits)

	c.Cleanup(ids)
	bm = c.AggregateAndReset()
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(c)

	ids.Add("test2")
	c.Cleanup(ids)
	bm = c.AggregateAndReset()
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)
}
