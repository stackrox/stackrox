package m58tom59

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

var (
	imageIntegrationBucket = []byte("imageintegrations")

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()

	migration = types.Migration{
		StartingSeqNum: 58,
		VersionAfter:   &storage.Version{SeqNum: 59},
		Run: func(databases *types.Databases) error {
			if err := migrateNodes(databases.BoltDB, databases.RocksDB); err != nil {
				return err
			}
			return migrateScanner(databases.BoltDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func bytesCopy(b []byte) []byte {
	newBytes := make([]byte, len(b))
	copy(newBytes, b)
	return newBytes
}

func httpsEndpointToGRPC(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	portIdx := strings.LastIndex(endpoint, ":")
	if portIdx != -1 {
		endpoint = endpoint[:portIdx]
	}
	return fmt.Sprintf("%s:8443", endpoint)
}

func migrateScanner(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var imageIntegration storage.ImageIntegration
			if err := proto.Unmarshal(v, &imageIntegration); err != nil {
				return err
			}
			if imageIntegration.GetType() != "clairify" {
				return nil
			}
			clairify := imageIntegration.GetClairify()
			clairify.GrpcEndpoint = httpsEndpointToGRPC(clairify.GetEndpoint())

			imageIntegration.Categories = append(imageIntegration.Categories, storage.ImageIntegrationCategory_NODE_SCANNER)

			newValue, err := proto.Marshal(&imageIntegration)
			if err != nil {
				return errors.Wrapf(err, "error marshalling external backup %s", k)
			}
			return bucket.Put(k, newValue)
		})
	})
}

func migrateNodes(boltDB *bolt.DB, rocksdb *gorocksdb.DB) error {
	writeBatch := gorocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	err := boltDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(nodeBucketName)
		if bucket == nil {
			log.WriteToStderr("no nodes bucket")
			return nil
		}
		c := bucket.Cursor()
		for clusterID, value := c.First(); clusterID != nil; clusterID, value = c.Next() {
			// bolt bucket references have nil values
			if value != nil {
				log.WriteToStderrf("key %s in node bucket has value %s", clusterID, value)
				continue
			}
			clusterKey := getClusterKey(clusterID)
			clusterKeyString := string(clusterKey)
			dackboxMappings := make(map[string]SortedKeys)

			relationshipKeys, err := rocksdb.Get(readOpts, getGraphKey(clusterKey))
			if err != nil {
				return err
			}

			if relationshipKeys.Exists() {
				ss, err := Unmarshal(relationshipKeys.Data())
				if err != nil {
					return err
				}
				dackboxMappings[clusterKeyString] = ss
			}

			nodeStoreBucket := bucket.Bucket(clusterID)
			if nodeStoreBucket == nil {
				continue
			}
			err = nodeStoreBucket.ForEach(func(k, v []byte) error {
				nodeKey := getNodeKey(k)

				// Write out the nodes into dackbox
				writeBatch.Put(nodeKey, bytesCopy(v))

				// Add the mappings between the objects to the map.
				dackboxMappings[clusterKeyString], _ = dackboxMappings[clusterKeyString].Insert(nodeKey)
				dackboxMappings[string(nodeKey)] = SortedCopy([][]byte{})
				return nil
			})
			if err != nil {
				return err
			}
			for from, tos := range dackboxMappings {
				writeBatch.Put(getGraphKey([]byte(from)), tos.Marshal())
			}
			if err := rocksdb.Write(writeOpts, writeBatch); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return boltDB.Update(func(tx *bolt.Tx) error {
		if tx.Bucket(nodeBucketName) != nil {
			return tx.DeleteBucket(nodeBucketName)
		}
		return nil
	})
}
