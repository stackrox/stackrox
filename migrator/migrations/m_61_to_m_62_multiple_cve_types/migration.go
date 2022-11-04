package m61tom62

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	rocksdb "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 100
)

var (
	migration = types.Migration{
		StartingSeqNum: 61,
		VersionAfter:   &storage.Version{SeqNum: 62},
		Run: func(databases *types.Databases) error {
			return migrateCVEs(databases.RocksDB)
		},
	}

	cveBucket = []byte("image_vuln")

	writeOpts = rocksdb.DefaultWriteOptions()
	readOpts  = rocksdb.DefaultIteratorOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateCVEs(db *gorocksdb.DB) error {
	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	it := db.NewIterator(readOpts)
	defer it.Close()

	for it.Seek(cveBucket); it.ValidForPrefix(cveBucket); it.Next() {
		key := it.Key().Copy()
		cveID := rocksdbmigration.GetIDFromPrefixedKey(cveBucket, key)

		var cve storage.CVE
		if err := proto.Unmarshal(it.Value().Data(), &cve); err != nil {
			return errors.Wrapf(err, "unmarshaling %s", cveID)
		}

		cve.Types = []storage.CVE_CVEType{cve.GetType()}
		cve.Type = storage.CVE_UNKNOWN_CVE

		data, err := proto.Marshal(&cve)
		if err != nil {
			return errors.Wrapf(err, "marshaling %s", cveID)
		}

		batch.Put(key, data)

		if batch.Count() == batchSize {
			if err := db.Write(writeOpts, batch); err != nil {
				return errors.Wrap(err, "writing CVEs to RocksDB")
			}
			batch.Clear()
		}
	}

	if batch.Count() != 0 {
		if err := db.Write(writeOpts, batch); err != nil {
			return errors.Wrap(err, "writing final batch of CVEs to RocksDB")
		}
	}

	return nil
}
