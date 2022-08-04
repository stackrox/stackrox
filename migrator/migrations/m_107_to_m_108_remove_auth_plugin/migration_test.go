package m107tom108

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

var (
	unmigratedPSs = []*storage.PermissionSet{
		{
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"AuthPlugin": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	unmigratedPSsAfterMigration = []*storage.PermissionSet{
		{
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	alreadyMigratedPSs = []*storage.PermissionSet{
		{
			Name:             "ps2",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
		{
			Name:             "ps3",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
	}
)

func TestPSAuthPluginMigration(t *testing.T) {
	db := testutils.DBForT(t)

	var psToUpsert []*storage.PermissionSet
	psToUpsert = append(psToUpsert, unmigratedPSs...)
	psToUpsert = append(psToUpsert, alreadyMigratedPSs...)

	require.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket(psBucket)
		if err != nil {
			return err
		}

		for _, ps := range psToUpsert {
			bytes, err := proto.Marshal(ps)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(ps.GetName()), bytes); err != nil {
				return err
			}
		}
		_, err = tx.CreateBucket(authPluginBucket)
		return err
	}))

	require.NoError(t, migratePS(db))

	var allPSsAfterMigration []*storage.PermissionSet

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(psBucket)
		if bucket == nil {
			return errors.New("bucket does not exist")
		}
		return bucket.ForEach(func(k, v []byte) error {
			ps := &storage.PermissionSet{}
			if err := proto.Unmarshal(v, ps); err != nil {
				return err
			}
			if string(k) != ps.GetName() {
				return errors.Errorf("Name mismatch: %s vs %s", k, ps.GetName())
			}
			allPSsAfterMigration = append(allPSsAfterMigration, ps)
			return nil
		})
	}))

	var expectedPSsAfterMigration []*storage.PermissionSet
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, unmigratedPSsAfterMigration...)
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, alreadyMigratedPSs...)

	assert.ElementsMatch(t, expectedPSsAfterMigration, allPSsAfterMigration)
}
