package bolt

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
)

func TestStore(t *testing.T) {
	t.Parallel()

	assert, require := testutils.AssertRequire(t)

	db, err := bolthelper.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(err)
	defer utils.IgnoreError(db.Close)

	store, err := NewBoltStore(db)
	require.NoError(err)

	whitelistResults := &storage.ProcessWhitelistResults{
		DeploymentId:      "BLAH",
		WhitelistStatuses: []*storage.ContainerNameAndWhitelistStatus{{ContainerName: "BLAHHH"}},
	}
	err = store.Upsert(whitelistResults)
	require.NoError(err)

	retrievedResults, exists, err := store.Get("BLAH")
	require.NoError(err)
	require.True(exists)
	assert.Equal(whitelistResults, retrievedResults)

	err = store.Delete("BLAH")
	require.NoError(err)

	retrievedResults, exists, err = store.Get("BLAH")
	require.NoError(err)
	require.False(exists)
	require.Nil(retrievedResults)
}
