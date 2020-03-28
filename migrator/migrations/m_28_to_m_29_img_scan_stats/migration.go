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
	imageBucketName     = []byte("imageBucket")
	listImageBucketName = []byte("images_list")
	migration           = types.Migration{
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
		defer batch.Cancel()
		for it.Seek(imageBucketName); it.ValidForPrefix(imageBucketName); it.Next() {
			if batch.Error() != nil {
				return batch.Error()
			}

			key := it.Item().KeyCopy([]byte{})
			err := it.Item().Value(func(v []byte) error {
				var image storage.Image
				if err := proto.Unmarshal(v, &image); err != nil {
					return errors.Wrapf(err, "unmarshal error for image: %s", key)
				}

				fillScanStats(&image)
				listImage := convertImageToListImage(&image)

				if err := writeImage(batch, key, image); err != nil {
					return err
				}
				return writeListImage(batch, key, *listImage)
			})
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

func writeImage(batch *badger.WriteBatch, key []byte, image storage.Image) error {
	data, err := proto.Marshal(&image)
	if err != nil {
		return errors.Wrapf(err, "marshal error for image: %s", key)
	}
	if err := batch.Set(key, data); err != nil {
		return errors.Wrapf(err, "error setting key/value in Badger for bucket %q", string(imageBucketName))
	}
	return nil
}

func writeListImage(batch *badger.WriteBatch, key []byte, listImage storage.ListImage) error {
	data, err := proto.Marshal(&listImage)
	if err != nil {
		return errors.Wrapf(err, "marshal error for list image: %s", key)
	}
	key = getKey(listImageBucketName, listImage.GetId())
	if err := batch.Set(key, data); err != nil {
		return errors.Wrapf(err, "error setting key/value in Badger for bucket %q", string(listImageBucketName))
	}
	return nil
}

func fillScanStats(i *storage.Image) {
	if i.GetScan() != nil {
		i.SetComponents = &storage.Image_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}

		var fixedByProvided bool
		var imageTopCVSS float32
		vulns := make(map[string]bool)
		for _, c := range i.GetScan().GetComponents() {
			var componentTopCVSS float32
			var hasVulns bool
			for _, v := range c.GetVulns() {
				hasVulns = true
				if _, ok := vulns[v.GetCve()]; !ok {
					vulns[v.GetCve()] = false
				}

				if v.GetCvss() > componentTopCVSS {
					componentTopCVSS = v.GetCvss()
				}

				if v.GetSetFixedBy() == nil {
					continue
				}

				fixedByProvided = true
				if v.GetFixedBy() != "" {
					vulns[v.GetCve()] = true
				}
			}

			if hasVulns {
				c.SetTopCvss = &storage.EmbeddedImageScanComponent_TopCvss{
					TopCvss: componentTopCVSS,
				}
			}

			if componentTopCVSS > imageTopCVSS {
				imageTopCVSS = componentTopCVSS
			}
		}

		i.SetCves = &storage.Image_Cves{
			Cves: int32(len(vulns)),
		}

		if len(vulns) > 0 {
			i.SetTopCvss = &storage.Image_TopCvss{
				TopCvss: imageTopCVSS,
			}
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

func convertImageToListImage(i *storage.Image) *storage.ListImage {
	listImage := &storage.ListImage{
		Id:          i.GetId(),
		Name:        i.GetName().GetFullName(),
		Created:     i.GetMetadata().GetV1().GetCreated(),
		LastUpdated: i.GetLastUpdated(),
	}

	if i.GetSetComponents() != nil {
		listImage.SetComponents = &storage.ListImage_Components{
			Components: i.GetComponents(),
		}
	}
	if i.GetSetCves() != nil {
		listImage.SetCves = &storage.ListImage_Cves{
			Cves: i.GetCves(),
		}
	}
	if i.GetSetFixable() != nil {
		listImage.SetFixable = &storage.ListImage_FixableCves{
			FixableCves: i.GetFixableCves(),
		}
	}
	return listImage
}

func getKey(bucketName []byte, id string) []byte {
	key := make([]byte, 0, len(bucketName)+len(id)+1)
	key = append(key, bucketName...)
	key = append(key, []byte("\x00")...)
	key = append(key, id...)

	return key
}

func init() {
	migrations.MustRegisterMigration(migration)
}
