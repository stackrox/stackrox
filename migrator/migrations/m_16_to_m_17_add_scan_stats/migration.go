package m16tom17

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var migration = types.Migration{
	StartingSeqNum: 16,
	VersionAfter:   storage.Version{SeqNum: 17},
	Run:            updateAllImages,
}

var (
	imagesBucketName = []byte("imageBucket")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updateAllImages(db *bolt.DB, _ *badger.DB) error {
	imagesBucket := bolthelpers.TopLevelRef(db, imagesBucketName)

	err := imagesBucket.Update(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			// Read current image.
			var image storage.Image
			if err := proto.Unmarshal(v, &image); err != nil {
				return errors.Wrap(err, "unmarshaling image")
			}

			// Fill in it's scan stats.
			fillScanStats(&image)

			// Write the new image.
			bytes, err := proto.Marshal(&image)
			if err != nil {
				return err
			}
			return b.Put(k, bytes)
		})
	})

	if err != nil {
		return err
	}

	return nil
}

// fillScanStats fills in the higher level stats from the scan data.
func fillScanStats(i *storage.Image) {
	if i.GetScan() != nil {
		i.SetComponents = &storage.Image_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}
		var numVulns int32
		var numFixableVulns int32
		var fixedByProvided bool
		for _, c := range i.GetScan().GetComponents() {
			numVulns += int32(len(c.GetVulns()))
			for _, v := range c.GetVulns() {
				if v.GetSetFixedBy() != nil {
					fixedByProvided = true
					if v.GetFixedBy() != "" {
						numFixableVulns++
					}
				}
			}
		}
		i.SetCves = &storage.Image_Cves{
			Cves: numVulns,
		}
		if numVulns == 0 || fixedByProvided {
			i.SetFixable = &storage.Image_FixableCves{
				FixableCves: numFixableVulns,
			}
		}
	}
}
