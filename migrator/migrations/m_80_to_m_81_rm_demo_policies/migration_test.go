package m80tom81

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/common/test"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestMigrationWithoutEditedPolicy(t *testing.T) {
	db := test.GetDBWithBucket(t, policyBucket)

	for _, c := range []struct {
		policyID   string
		policyName string
		file       string
	}{
		{
			policyID:   "5a90a571-58e7-4ed5-a2fa-2dbe83e649ba",
			policyName: "DockerHub NGINX 1.10",
			file:       "policies/nginx.json",
		},
		{
			policyID:   "9d1ebb72-7b76-4a21-a058-3d8fdf451037",
			policyName: "Heartbleed: CVE-2014-0160",
			file:       "policies/heartbleed.json",
		},
		{
			policyID:   "2b251b91-fd41-4a71-ad01-586c385714ba",
			policyName: "Shellshock: Multiple CVEs",
			file:       "policies/shellshock.json",
		},
	} {
		t.Run("", func(t *testing.T) {
			policyToRm, err := policymigrationhelper.ReadPolicyFromFile(policiesFS, c.file)
			require.NoError(t, err)

			err = db.Update(func(tx *bolt.Tx) error {
				data, err := policyToRm.Marshal()
				if err != nil {
					return err
				}
				return tx.Bucket(policyBucket).Put([]byte(c.policyID), data)
			})
			require.NoError(t, err)

			// Verify default policies exists.
			err = db.View(func(tx *bolt.Tx) error {
				require.NotNil(t, tx.Bucket(policyBucket).Get([]byte(c.policyID)))
				return nil
			})
			require.NoError(t, err)

			// Run migration
			assert.NoError(t, rmDemoPolicies(db))

			// Verify default policies is removed.
			err = db.View(func(tx *bolt.Tx) error {
				assert.Nil(t, tx.Bucket(policyBucket).Get([]byte(c.policyID)))
				return nil
			})
			assert.NoError(t, err)
		})
	}
}

func TestMigrationWithEditedPolicy(t *testing.T) {
	db := test.GetDBWithBucket(t, policyBucket)
	nginxPolicyID := "5a90a571-58e7-4ed5-a2fa-2dbe83e649ba"

	err := db.Update(func(tx *bolt.Tx) error {
		policy := &storage.Policy{
			Id:   nginxPolicyID,
			Name: "DockerHub NGINX 1.10",
		}
		data, err := policy.Marshal()
		if err != nil {
			return err
		}
		return tx.Bucket(policyBucket).Put([]byte(nginxPolicyID), data)
	})
	require.NoError(t, err)

	// Verify default policies exists.
	err = db.View(func(tx *bolt.Tx) error {
		require.NotNil(t, tx.Bucket(policyBucket).Get([]byte(nginxPolicyID)))
		return nil
	})
	require.NoError(t, err)

	// Run migration
	assert.NoError(t, rmDemoPolicies(db))

	// Verify default policies is not removed.
	err = db.View(func(tx *bolt.Tx) error {
		assert.NotNil(t, tx.Bucket(policyBucket).Get([]byte(nginxPolicyID)))
		return nil
	})
	assert.NoError(t, err)
}
