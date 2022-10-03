package manager

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPv6Sort(t *testing.T) {
	ipv6Slice := []uint64{
		14, 2,
		3, 100,
		100, 3,
		1, 1000,
		14, 3,
		14, 1,
	}

	sort.Sort(sortableIPv6Slice(ipv6Slice))

	expectedSortedSlice := []uint64{
		1, 1000,
		3, 100,
		14, 1,
		14, 2,
		14, 3,
		100, 3,
	}

	assert.Equal(t, expectedSortedSlice, ipv6Slice)
}

func TestPublicIPsManager(t *testing.T) {
	mgr := newPublicIPsManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go mgr.Run(ctx, clusterentities.NewStore())

	vs := mgr.PublicIPsProtoStream().Iterator(true)

	assert.Nil(t, vs.Value())
	assert.Nil(t, vs.TryNext())
	assert.False(t, mgr.publicIPsUpdateSig.IsDone())

	mgr.OnAdded(net.ParseIP("4.4.4.4"))

	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Addresses(), 1)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())
	assert.False(t, mgr.publicIPsUpdateSig.IsDone())

	mgr.OnAdded(net.ParseIP("8.8.8.8"))

	require.True(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	vs = vs.TryNext()
	require.NotNil(t, vs)
	assert.Len(t, vs.Value().GetIpv4Addresses(), 2)

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())
	assert.False(t, mgr.publicIPsUpdateSig.IsDone())

	mgr.OnRemoved(net.ParseIP("4.4.4.4"))

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())
	assert.False(t, mgr.publicIPsUpdateSig.IsDone())

	mgr.OnAdded(net.ParseIP("4.4.4.4"))

	assert.False(t, concurrency.WaitWithTimeout(vs, 100*time.Millisecond))
	assert.Nil(t, vs.TryNext())
	assert.False(t, mgr.publicIPsUpdateSig.IsDone())
}
