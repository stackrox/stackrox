package m35tom36

import (
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	unmigratedClusters = []*storage.Cluster{
		{
			Id:   "0",
			Name: "cluster0",
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: true,
			},
		},
		{
			Id:   "1",
			Name: "cluster1",
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: false,
			},
			CollectionMethod: storage.CollectionMethod_KERNEL_MODULE,
			DynamicConfig:    &storage.DynamicClusterConfig{},
		},
	}

	unmigratedClustersAfterMigration = []*storage.Cluster{
		{
			Id:   "0",
			Name: "cluster0",
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: true,
			},
			CollectionMethod: storage.CollectionMethod_NO_COLLECTION,
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					TimeoutSeconds: defaultAdmissionControllerTimeout,
				},
			},
		},
		{
			Id:   "1",
			Name: "cluster1",
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: false,
			},
			CollectionMethod: storage.CollectionMethod_KERNEL_MODULE,
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					TimeoutSeconds: defaultAdmissionControllerTimeout,
				},
			},
		},
	}

	alreadyMigratedClusters = []*storage.Cluster{
		{
			Id:   "2",
			Name: "cluster2",
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: true,
			},
			CollectionMethod: storage.CollectionMethod_EBPF,
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					TimeoutSeconds: 3,
				},
			},
		},
		{
			Id:   "3",
			Name: "cluster3",
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: false,
			},
			CollectionMethod: storage.CollectionMethod_NO_COLLECTION,
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					TimeoutSeconds: defaultAdmissionControllerTimeout,
				},
			},
		},
	}
)

func TestClusterNormalizationMigration(t *testing.T) {
	db := testutils.DBForT(t)

	var clustersToUpsert []*storage.Cluster
	clustersToUpsert = append(clustersToUpsert, unmigratedClusters...)
	clustersToUpsert = append(clustersToUpsert, alreadyMigratedClusters...)

	require.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket(clustersBucket)
		if err != nil {
			return err
		}

		for _, cluster := range clustersToUpsert {
			bytes, err := proto.Marshal(cluster)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(cluster.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	}))

	require.NoError(t, addDynamicClusterConfig(db))

	var allClustersAfterMigration []*storage.Cluster

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		if bucket == nil {
			return errors.New("bucket does not exist")
		}
		return bucket.ForEach(func(k, v []byte) error {
			cluster := &storage.Cluster{}
			if err := proto.Unmarshal(v, cluster); err != nil {
				return err
			}
			if string(k) != cluster.GetId() {
				return errors.Errorf("ID mismatch: %s vs %s", k, cluster.GetId())
			}
			allClustersAfterMigration = append(allClustersAfterMigration, cluster)
			return nil
		})
	}))

	var expectedClustersAfterMigration []*storage.Cluster
	expectedClustersAfterMigration = append(expectedClustersAfterMigration, unmigratedClustersAfterMigration...)
	expectedClustersAfterMigration = append(expectedClustersAfterMigration, alreadyMigratedClusters...)

	assert.ElementsMatch(t, expectedClustersAfterMigration, allClustersAfterMigration)
}
