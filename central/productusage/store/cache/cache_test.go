package cache

import (
	"testing"

	"github.com/stackrox/rox/central/productusage/source"
	"github.com/stackrox/rox/central/productusage/source/mocks"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func makeSource(ctrl *gomock.Controller, n int64, c int64) source.SecuredUnitsSource {
	s := mocks.NewMockSecuredUnitsSource(ctrl)
	s.EXPECT().GetNodeCount().AnyTimes().Return(n)
	s.EXPECT().GetCpuCapacity().AnyTimes().Return(c)
	return s
}

func inject(ctrl *gomock.Controller, c Cache) {
	c.UpdateUsage("test1", makeSource(ctrl, 1, 10))
	c.UpdateUsage("test2", makeSource(ctrl, 2, 20))
}

func TestCleanupCurrent(t *testing.T) {
	c := NewCache()
	ctrl := gomock.NewController(t)

	inject(ctrl, c)

	ids := set.NewStringSet()
	c.Cleanup(ids)
	bm := c.GetCurrent()
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(ctrl, c)

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

	inject(ctrl, c)

	c.Cleanup(ids)
	bm = c.GetCurrent()
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)
}

func TestAggregateAndFlush(t *testing.T) {
	c := NewCache()
	ctrl := gomock.NewController(t)

	inject(ctrl, c)
	bm := c.AggregateAndFlush()
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)

	ids := set.NewStringSet()
	c.Cleanup(ids)
	bm = c.AggregateAndFlush()
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(ctrl, c)

	ids.Add("test1")
	c.Cleanup(ids)
	bm = c.AggregateAndFlush()
	assert.Equal(t, int64(1), bm.NumNodes)
	assert.Equal(t, int64(10), bm.NumCpuUnits)

	c.Cleanup(ids)
	bm = c.AggregateAndFlush()
	assert.Equal(t, int64(0), bm.NumNodes)
	assert.Equal(t, int64(0), bm.NumCpuUnits)

	inject(ctrl, c)

	ids.Add("test2")
	c.Cleanup(ids)
	bm = c.AggregateAndFlush()
	assert.Equal(t, int64(3), bm.NumNodes)
	assert.Equal(t, int64(30), bm.NumCpuUnits)
}
