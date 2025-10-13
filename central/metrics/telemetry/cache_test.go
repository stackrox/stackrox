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

	cache.Set("cluster-1", &central.ClusterMetrics{CpuCapacity: 10, NodeCount: 1})
	protoassert.Equal(t, &central.ClusterMetrics{CpuCapacity: 10, NodeCount: 1}, cache.Sum())
	assert.Equal(t, 1, cache.Len())

	cache.Set("cluster-2", &central.ClusterMetrics{CpuCapacity: 20, NodeCount: 2})
	protoassert.Equal(t, &central.ClusterMetrics{CpuCapacity: 30, NodeCount: 3}, cache.Sum())
	assert.Equal(t, 2, cache.Len())

	cache.Set("cluster-1", &central.ClusterMetrics{CpuCapacity: 20, NodeCount: 3})
	protoassert.Equal(t, &central.ClusterMetrics{CpuCapacity: 40, NodeCount: 5}, cache.Sum())
	assert.Equal(t, 2, cache.Len())

	cache.Delete("cluster-1")
	protoassert.Equal(t, &central.ClusterMetrics{CpuCapacity: 20, NodeCount: 2}, cache.Sum())
	assert.Equal(t, 1, cache.Len())

	cache.Delete("cluster-3")
	protoassert.Equal(t, &central.ClusterMetrics{CpuCapacity: 20, NodeCount: 2}, cache.Sum())
	assert.Equal(t, 1, cache.Len())
}
