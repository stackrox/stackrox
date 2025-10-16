package telemetry

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoMetric(t *testing.T) {
	cache := newMetricsCache()
	require.NotNil(t, cache)

	cm := &central.ClusterMetrics{}
	cm.SetCpuCapacity(10)
	cm.SetNodeCount(1)
	cache.Set("cluster-1", cm)
	cm2 := &central.ClusterMetrics{}
	cm2.SetCpuCapacity(10)
	cm2.SetNodeCount(1)
	protoassert.Equal(t, cm2, cache.Sum())
	assert.Equal(t, 1, cache.Len())

	cm3 := &central.ClusterMetrics{}
	cm3.SetCpuCapacity(20)
	cm3.SetNodeCount(2)
	cache.Set("cluster-2", cm3)
	cm4 := &central.ClusterMetrics{}
	cm4.SetCpuCapacity(30)
	cm4.SetNodeCount(3)
	protoassert.Equal(t, cm4, cache.Sum())
	assert.Equal(t, 2, cache.Len())

	cm5 := &central.ClusterMetrics{}
	cm5.SetCpuCapacity(20)
	cm5.SetNodeCount(3)
	cache.Set("cluster-1", cm5)
	cm6 := &central.ClusterMetrics{}
	cm6.SetCpuCapacity(40)
	cm6.SetNodeCount(5)
	protoassert.Equal(t, cm6, cache.Sum())
	assert.Equal(t, 2, cache.Len())

	cache.Delete("cluster-1")
	cm7 := &central.ClusterMetrics{}
	cm7.SetCpuCapacity(20)
	cm7.SetNodeCount(2)
	protoassert.Equal(t, cm7, cache.Sum())
	assert.Equal(t, 1, cache.Len())

	cache.Delete("cluster-3")
	cm8 := &central.ClusterMetrics{}
	cm8.SetCpuCapacity(20)
	cm8.SetNodeCount(2)
	protoassert.Equal(t, cm8, cache.Sum())
	assert.Equal(t, 1, cache.Len())
}
