package m71tom72

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

var (
	migration = types.Migration{
		StartingSeqNum: 71,
		VersionAfter:   &storage.Version{SeqNum: 72},
		Run: func(databases *types.Databases) error {
			return deleteNamespaceSACBucketAndEdges(databases.RocksDB)
		},
	}
)

func deleteNamespaceSACBucketAndEdges(db *gorocksdb.DB) error {
	if err := deleteNamespaceSACBucket(db); err != nil {
		return err
	}
	if err := deleteNamespaceSACEdges(db); err != nil {
		return err
	}
	return nil
}

func deleteNamespaceSACBucket(db *gorocksdb.DB) error {
	scanOpts := gorocksdb.NewDefaultReadOptions()
	defer scanOpts.Destroy()

	scanOpts.SetPrefixSameAsStart(true)
	scanOpts.SetFillCache(false)

	it := db.NewIterator(scanOpts)
	defer it.Close()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	nsSACPrefix := prefixKey(getGraphKey(nsSACBucketName), nil)
	for it.Seek(prefixKey(getGraphKey(nsSACBucketName), nil)); it.ValidForPrefix(nsSACPrefix); it.Next() {
		batch.Delete(it.Key().Data())
	}

	writeOpts := gorocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()
	if err := db.Write(writeOpts, batch); err != nil {
		return errors.Wrap(err, "flushing key deletion batch")
	}
	return nil
}

func deleteNamespaceSACEdges(db *gorocksdb.DB) error {
	scanOpts := gorocksdb.NewDefaultReadOptions()
	defer scanOpts.Destroy()

	scanOpts.SetPrefixSameAsStart(true)
	scanOpts.SetFillCache(false)

	it := db.NewIterator(scanOpts)
	defer it.Close()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	startKey := prefixKey(nsSACBucketName, nil)
	endKey := append([]byte{}, startKey...)
	endKey[len(endKey)-1]++ // smallest key larger than any element with prefix startKey

	nsPrefix := prefixKey(getGraphKey(nsBucketName), nil)
	for it.Seek(nsPrefix); it.ValidForPrefix(nsPrefix); it.Next() {
		sks, err := Unmarshal(it.Value().Data())
		if err != nil {
			return errors.Wrap(err, "unmarshaling sorted keys")
		}

		startPos, _ := sks.positionOf(startKey)
		if startPos == len(sks) {
			continue
		}

		endPos, _ := sks[startPos:].positionOf(endKey)
		sks = append(sks[:startPos], sks[startPos+endPos:]...)
		batch.Put(it.Key().Data(), sks.Marshal())
	}

	writeOpts := gorocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	if err := db.Write(writeOpts, batch); err != nil {
		return errors.Wrap(err, "flushing edge deletion batch")
	}

	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
