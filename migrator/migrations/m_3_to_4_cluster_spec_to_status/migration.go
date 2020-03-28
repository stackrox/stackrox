package m3to4

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	clusterBucketName       = []byte("clusters")
	clusterStatusBucketName = []byte("cluster_status")

	clusterStatusMigration = types.Migration{
		StartingSeqNum: 3,
		VersionAfter:   storage.Version{SeqNum: 4},
		Run: func(db *bolt.DB, _ *badger.DB) error {
			clusterBucket := bolthelpers.TopLevelRef(db, clusterBucketName)
			clusterStatusBucket, err := bolthelpers.TopLevelRefWithCreateIfNotExists(db, clusterStatusBucketName)
			if err != nil {
				return err
			}

			clusterStatuses := make(map[string]*storage.ClusterStatus)

			err = clusterBucket.View(func(b *bolt.Bucket) error {
				return b.ForEach(func(k, v []byte) error {
					cluster := new(storage.Cluster)
					err := proto.Unmarshal(v, cluster)
					if err != nil {
						return errors.Wrap(err, "proto umarshaling failed")
					}
					if cluster.GetDEPRECATEDProviderMetadata() != nil || cluster.GetDEPRECATEDOrchestratorMetadata() != nil {
						clusterStatuses[cluster.GetId()] = &storage.ClusterStatus{
							ProviderMetadata:     cluster.GetDEPRECATEDProviderMetadata(),
							OrchestratorMetadata: cluster.GetDEPRECATEDOrchestratorMetadata(),
						}
					}
					return nil
				})
			})
			if err != nil {
				return errors.Wrap(err, "failed to read existing clusters into memory")
			}

			err = clusterBucket.Update(func(b *bolt.Bucket) error {
				for id := range clusterStatuses {
					cluster := new(storage.Cluster)
					bytes := b.Get([]byte(id))
					err := proto.Unmarshal(bytes, cluster)
					if err != nil {
						return errors.Wrap(err, "proto unmarshaling failed")
					}
					cluster.DEPRECATEDOrchestratorMetadata = nil
					cluster.DEPRECATEDProviderMetadata = nil
					newBytes, err := proto.Marshal(cluster)
					if err != nil {
						return errors.Wrap(err, "proto marshaling failed")
					}
					if err := b.Put([]byte(id), newBytes); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "failed to insert updated clusters")
			}

			return clusterStatusBucket.Update(func(b *bolt.Bucket) error {
				for id, status := range clusterStatuses {
					existingStatus := new(storage.ClusterStatus)
					existingStatusBytes := b.Get([]byte(id))
					if existingStatusBytes != nil {
						if err := proto.Unmarshal(existingStatusBytes, existingStatus); err != nil {
							return errors.Wrap(err, "unmarshaling existing status")
						}
					}
					existingStatus.ProviderMetadata = status.ProviderMetadata
					existingStatus.OrchestratorMetadata = status.OrchestratorMetadata
					marshalled, err := proto.Marshal(existingStatus)
					if err != nil {
						return errors.Wrapf(err, "marshaling status %+v", existingStatus)
					}
					if err := b.Put([]byte(id), []byte(marshalled)); err != nil {
						return errors.Wrap(err, "inserting into bolt")
					}
				}
				return nil
			})
		},
	}
)

func init() {
	migrations.MustRegisterMigration(clusterStatusMigration)
}
