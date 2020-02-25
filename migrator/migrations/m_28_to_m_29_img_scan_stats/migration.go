package m28to29

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	imageBucketName = []byte("image\x00")
	migration       = types.Migration{
		StartingSeqNum: 28,
		VersionAfter:   storage.Version{SeqNum: 29},
		Run:            rewriteImagesWithCorrectScanStats,
	}
)

func rewriteImagesWithCorrectScanStats(_ *bolt.DB, badgerDB *badger.DB) error {
	return badgerDB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		batch := badgerDB.NewWriteBatch()
		for it.Seek(imageBucketName); it.ValidForPrefix(imageBucketName); it.Next() {
			if batch.Error() != nil {
				return batch.Error()
			}

			key := it.Item().Key()
			err := it.Item().Value(func(v []byte) error {
				var image storage.Image
				if err := proto.Unmarshal(v, &image); err != nil {
					return errors.Wrapf(err, "unmarshal error for image: %s", key)
				}

				fillScanStats(&image)

				data, err := proto.Marshal(&image)
				if err != nil {
					return errors.Wrapf(err, "marshal error for image: %s", key)
				}

				if err := batch.Set(key, data); err != nil {
					return errors.Wrapf(err, "error setting key/value in Badger for bucket %q", string(imageBucketName))
				}
				return nil
			})
			defer batch.Cancel()
			if err != nil {
				return err
			}
		}
		if err := batch.Flush(); err != nil {
			return errors.Wrapf(err, "error flushing BadgerDB for bucket %q", string(imageBucketName))
		}
		return nil
	})
}

func fillScanStats(i *storage.Image) {
	if i.GetScan() != nil {
		i.SetComponents = &storage.Image_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}

		var fixedByProvided bool
		vulns := make(map[string]bool)
		for _, c := range i.GetScan().GetComponents() {
			for _, v := range c.GetVulns() {
				if _, ok := vulns[v.GetCve()]; !ok {
					vulns[v.GetCve()] = false
				}

				if v.GetSetFixedBy() == nil {
					continue
				}

				fixedByProvided = true
				if v.GetFixedBy() != "" {
					vulns[v.GetCve()] = true
				}
			}
		}

		i.SetCves = &storage.Image_Cves{
			Cves: int32(len(vulns)),
		}
		if int32(len(vulns)) == 0 || fixedByProvided {
			var numFixableVulns int32
			for _, fixable := range vulns {
				if fixable {
					numFixableVulns++
				}
			}
			i.SetFixable = &storage.Image_FixableCves{
				FixableCves: numFixableVulns,
			}
		}
	}
}

func init() {
	migrations.MustRegisterMigration(migration)
}
