package m68tom69

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

type permission struct {
	resource string
	access   storage.Access
}

var (
	unmigratedRoles = []*storage.Role{
		{
			Name:         "role0",
			GlobalAccess: storage.Access_READ_WRITE_ACCESS,
		},
		{
			Name:         "role1",
			GlobalAccess: storage.Access_READ_ACCESS,
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	unmigratedRolesAfterMigration = []*storage.Role{
		{
			Name:             "role0",
			ResourceToAccess: allResourcesWithAccess(storage.Access_READ_WRITE_ACCESS),
		},
		{
			Name:             "role1",
			ResourceToAccess: allResourcesWithAccess(storage.Access_READ_ACCESS, permission{"Image", storage.Access_READ_WRITE_ACCESS}),
		},
	}

	alreadyMigratedRoles = []*storage.Role{
		{
			Name:             "role2",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
		{
			Name:             "role3",
			ResourceToAccess: allResourcesWithAccess(storage.Access_READ_ACCESS),
		},
	}
)

func allResourcesWithAccess(access storage.Access, overrides ...permission) map[string]storage.Access {
	resources := make(map[string]storage.Access, len(AllResources))
	for _, v := range AllResources {
		resources[v] = access
	}
	for _, override := range overrides {
		resources[override.resource] = override.access
	}
	return resources
}

func TestRolesGlobalAccessMigration(t *testing.T) {
	db := testutils.DBForT(t)

	var rolesToUpsert []*storage.Role
	rolesToUpsert = append(rolesToUpsert, unmigratedRoles...)
	rolesToUpsert = append(rolesToUpsert, alreadyMigratedRoles...)

	require.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket(rolesBucket)
		if err != nil {
			return err
		}

		for _, role := range rolesToUpsert {
			bytes, err := proto.Marshal(role)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(role.GetName()), bytes); err != nil {
				return err
			}
		}
		return nil
	}))

	require.NoError(t, migrateRoles(db))

	var allRolesAfterMigration []*storage.Role

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		if bucket == nil {
			return errors.New("bucket does not exist")
		}
		return bucket.ForEach(func(k, v []byte) error {
			role := &storage.Role{}
			if err := proto.Unmarshal(v, role); err != nil {
				return err
			}
			if string(k) != role.GetName() {
				return errors.Errorf("Name mismatch: %s vs %s", k, role.GetName())
			}
			allRolesAfterMigration = append(allRolesAfterMigration, role)
			return nil
		})
	}))

	var expectedRolesAfterMigration []*storage.Role
	expectedRolesAfterMigration = append(expectedRolesAfterMigration, unmigratedRolesAfterMigration...)
	expectedRolesAfterMigration = append(expectedRolesAfterMigration, alreadyMigratedRoles...)

	assert.ElementsMatch(t, expectedRolesAfterMigration, allRolesAfterMigration)
}
