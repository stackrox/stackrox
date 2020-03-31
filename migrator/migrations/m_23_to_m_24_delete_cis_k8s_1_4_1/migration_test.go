package m23to24

import (
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	cases := []struct {
		runResults *storage.ComplianceRunResults
		deleted    bool
	}{
		{
			runResults: &storage.ComplianceRunResults{
				RunMetadata: &storage.ComplianceRunMetadata{
					ClusterId:  "1",
					StandardId: "CIS_Kubernetes_v1_4_1",
				},
			},
			deleted: true,
		},
		{
			runResults: &storage.ComplianceRunResults{
				RunMetadata: &storage.ComplianceRunMetadata{
					ClusterId:  "2",
					StandardId: "CIS_Kubernetes_v1_5",
				},
			},
			deleted: false,
		},
	}

	db, err := bolthelpers.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)

	createComplianceRunResultsBucket(t, db)

	for _, c := range cases {
		fillResultsData(t, db, c.runResults)
	}

	require.NoError(t, deleteUnsupportedComplianceStandards(&types.Databases{BoltDB: db}))

	for _, c := range cases {
		validateMigration(t, db, c.runResults, c.deleted)
	}
}

func createComplianceRunResultsBucket(t *testing.T, db *bbolt.DB) {
	assert.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(complianceResultsBucketName)
		return err
	}))
}

func fillResultsData(t *testing.T, db *bbolt.DB, runResults *storage.ComplianceRunResults) {
	assert.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		resultsBucket := tx.Bucket(complianceResultsBucketName)
		clusterID := runResults.GetRunMetadata().GetClusterId()
		clusterBucket, err := resultsBucket.CreateBucketIfNotExists([]byte(clusterID))
		if err != nil {
			return errors.Wrapf(err, "creating bucket for cluster %q", clusterID)
		}

		standardID := runResults.GetRunMetadata().GetStandardId()
		_, err = clusterBucket.CreateBucketIfNotExists([]byte(standardID))
		if err != nil {
			return errors.Wrapf(err, "creating bucket for standard %q", clusterID)
		}

		return nil
	}))
}

func validateMigration(t *testing.T, db *bbolt.DB, runResults *storage.ComplianceRunResults, deleted bool) {
	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		resultsBucket := tx.Bucket(complianceResultsBucketName)
		clusterBucket := resultsBucket.Bucket([]byte(runResults.GetRunMetadata().GetClusterId()))
		standardBucket := clusterBucket.Bucket([]byte(runResults.GetRunMetadata().GetStandardId()))

		if deleted {
			assert.Nil(t, standardBucket)
		} else {
			assert.NotNil(t, standardBucket)
		}

		return nil
	}))
}
