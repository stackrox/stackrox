package rocksdb

import (
	"testing"

	"github.com/stackrox/stackrox/central/networkgraph/flow/datastore/internal/store/testcommon"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	db, err := rocksdb.NewTemp(t.Name())
	require.NoError(t, err)

	store := NewClusterStore(db)
	flowSuite := testcommon.NewFlowStoreTest(store)
	suite.Run(t, flowSuite)
	rocksdbtest.TearDownRocksDB(db)
}
