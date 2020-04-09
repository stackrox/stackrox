package globalstore

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalStoreImpl(t *testing.T) {
	bolt := testutils.DBForT(t)
	gs := NewGlobalStore(bolt)
	ns, err := gs.GetClusterNodeStore("clusterid", true)
	require.NoError(t, err)

	err = ns.UpsertNode(&storage.Node{
		Id: "nodeid",
	})
	require.NoError(t, err)

	count, err := ns.CountNodes()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	require.NoError(t, gs.RemoveClusterNodeStores("clusterid"))

	count, err = gs.CountAllNodes()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
