package m22to23

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var complianceResultsBucketName = []byte("compliance-run-results")

var migration = types.Migration{
	StartingSeqNum: 22,
	VersionAfter:   storage.Version{SeqNum: 23},
	Run:            deleteUnsupportedComplianceStandards,
}

func deleteUnsupportedComplianceStandards(databases *types.Databases) error {
	db := databases.BoltDB

	err := db.Update(func(tx *bolt.Tx) error {
		complianceResultsBucket := tx.Bucket(complianceResultsBucketName)
		if complianceResultsBucket == nil {
			return nil
		}

		return deleteUnsupportedStandardRunResults(complianceResultsBucket)
	})
	return err
}

func deleteUnsupportedStandardRunResults(resultsBucket *bolt.Bucket) error {
	return resultsBucket.ForEach(func(clusterKey, _ []byte) error {
		clusterBucket := resultsBucket.Bucket(clusterKey)
		if clusterBucket == nil {
			return nil
		}

		return deleteUnsupportedStandardRunResultsForCluster(clusterBucket)
	})
}

func deleteUnsupportedStandardRunResultsForCluster(clusterBucket *bolt.Bucket) error {
	return clusterBucket.ForEach(func(standardKey, _ []byte) error {
		if !standardToDelete(string(standardKey)) {
			return nil
		}

		standardBucket := clusterBucket.Bucket(standardKey)
		if standardBucket == nil {
			return nil
		}

		return clusterBucket.DeleteBucket(standardKey)
	})
}

func standardToDelete(id string) bool {
	return id == "CIS_Docker_v1_1_0" || id == "CIS_Kubernetes_v1_2_0"
}

func init() {
	migrations.MustRegisterMigration(migration)
}
