package badger

import (
	"os"
	"testing"

	"github.com/stackrox/rox/central/networkflow/datastore/internal/store/testcommon"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestFlowStore(t *testing.T) {
	db, path, err := badgerhelper.NewTemp(t.Name())
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(path) }()
	defer utils.IgnoreError(db.Close)

	store := NewClusterStore(db)
	flowSuite := testcommon.NewFlowStoreTest(store)
	suite.Run(t, flowSuite)
}
