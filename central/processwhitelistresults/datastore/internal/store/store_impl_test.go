package store

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

	store, err := New(db)
	require.NoError(err)

	whitelistResults := &storage.ProcessWhitelistResults{
		DeploymentId:      "BLAH",
		WhitelistStatuses: []*storage.ContainerNameAndWhitelistStatus{{ContainerName: "BLAHHH"}},
	}
	err = store.UpsertWhitelistResults(whitelistResults)
	require.NoError(err)

	retrievedResults, err := store.GetWhitelistResults("BLAH")
	require.NoError(err)
	assert.Equal(whitelistResults, retrievedResults)

	err = store.DeleteWhitelistResults("BLAH")
	require.NoError(err)

	retrievedResults, err = store.GetWhitelistResults("BLAH")
	require.NoError(err)
	require.Nil(retrievedResults)
}
