package m3to4

import (
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
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
						return fmt.Errorf("proto umarshaling failed: %v", err)
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
				return fmt.Errorf("failed to read existing clusters into memory: %v", err)
			}

			err = clusterBucket.Update(func(b *bolt.Bucket) error {
				for id := range clusterStatuses {
					cluster := new(storage.Cluster)
					bytes := b.Get([]byte(id))
					err := proto.Unmarshal(bytes, cluster)
					if err != nil {
						return fmt.Errorf("proto unmarshaling failed: %v", err)
					}
					cluster.DEPRECATEDOrchestratorMetadata = nil
					cluster.DEPRECATEDProviderMetadata = nil
					newBytes, err := proto.Marshal(cluster)
					if err != nil {
						return fmt.Errorf("proto marshaling failed: %v", err)
					}
					if err := b.Put([]byte(id), newBytes); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to insert updated clusters: %v", err)
			}

			return clusterStatusBucket.Update(func(b *bolt.Bucket) error {
				for id, status := range clusterStatuses {
					existingStatus := new(storage.ClusterStatus)
					existingStatusBytes := b.Get([]byte(id))
					if existingStatusBytes != nil {
						if err := proto.Unmarshal(existingStatusBytes, existingStatus); err != nil {
							return fmt.Errorf("unmarshaling existing status: %v", err)
						}
					}
					existingStatus.ProviderMetadata = status.ProviderMetadata
					existingStatus.OrchestratorMetadata = status.OrchestratorMetadata
					marshalled, err := proto.Marshal(existingStatus)
					if err != nil {
						return fmt.Errorf("marshaling status %+v: %v", existingStatus, err)
					}
					if err := b.Put([]byte(id), []byte(marshalled)); err != nil {
						return fmt.Errorf("inserting into bolt: %v", err)
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
