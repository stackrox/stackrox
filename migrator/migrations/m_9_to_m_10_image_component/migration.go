package m9to10

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var migration = types.Migration{
	StartingSeqNum: 9,
	VersionAfter:   storage.Version{SeqNum: 10},
	Run:            migrateImageComponents,
}

var (
	imageBucket = []byte("imageBucket")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func stitchImageComponents(image *storage.Image) bool {
	if image.GetMetadata() == nil || image.GetScan() == nil {
		return false
	}

	var components []*storage.ImageScanComponent
	for i, l := range image.GetMetadata().GetV1().GetLayers() {
		for _, c := range l.DEPRECATEDComponents {
			c.HasLayerIndex = &storage.ImageScanComponent_LayerIndex{
				LayerIndex: int32(i),
			}
			components = append(components, c)
		}
		l.DEPRECATEDComponents = nil
	}
	if len(components) == 0 {
		return false
	}
	image.Scan.Components = components
	return true
}

func migrateImageComponents(db *bolt.DB, _ *badger.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var img storage.Image
			if err := proto.Unmarshal(v, &img); err != nil {
				return err
			}
			if changed := stitchImageComponents(&img); !changed {
				return nil
			}
			newData, err := proto.Marshal(&img)
			if err != nil {
				return err
			}
			return bucket.Put(k, newData)
		})
	})
}
