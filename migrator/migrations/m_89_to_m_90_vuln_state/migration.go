package m89tom90

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/dackboxhelpers"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	imageCVEEdgePrefix = []byte("image_to_cve")
	cvePrefix          = []byte("image_vuln")

	migration = types.Migration{
		StartingSeqNum: 89,
		VersionAfter:   &storage.Version{SeqNum: 90},
		Run:            updateImageCVEEdgesWithVulnState,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updateImageCVEEdgesWithVulnState(databases *types.Databases) error {
	suppressedCVEs, err := getSuppressedCVEs(databases.RocksDB)
	if err != nil {
		return err
	}

	it := databases.RocksDB.NewIterator(readOpts)
	defer it.Close()

	wb := gorocksdb.NewWriteBatch()
	for it.Seek(imageCVEEdgePrefix); it.ValidForPrefix(imageCVEEdgePrefix); it.Next() {
		key := it.Key().Copy()
		id := rocksdbmigration.GetIDFromPrefixedKey(imageCVEEdgePrefix, key)
		edgeID, err := dackboxhelpers.FromString(string(id))
		if err != nil {
			return err
		}

		if _, ok := suppressedCVEs[edgeID.ChildID]; !ok {
			continue
		}

		imageCVEEdge := &storage.ImageCVEEdge{}
		if err := proto.Unmarshal(it.Value().Data(), imageCVEEdge); err != nil {
			return errors.Wrapf(err, "unmarshaling image-cve edge %s", edgeID)
		}
		imageCVEEdge.State = storage.VulnerabilityState_DEFERRED

		newData, err := proto.Marshal(imageCVEEdge)
		if err != nil {
			return errors.Wrapf(err, "marshaling image-cve edge %s", key)
		}
		wb.Put(key, newData)

		if wb.Count() == batchSize {
			if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}

	if wb.Count() != 0 {
		if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}

func getSuppressedCVEs(db *gorocksdb.DB) (map[string]struct{}, error) {
	it := db.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()

	cves := make(map[string]struct{})
	for it.Seek(cvePrefix); it.ValidForPrefix(cvePrefix); it.Next() {
		id := rocksdbmigration.GetIDFromPrefixedKey(cvePrefix, it.Key().Copy())
		cve := &storage.CVE{}
		if err := proto.Unmarshal(it.Value().Data(), cve); err != nil {
			return nil, errors.Wrapf(err, "Failed to unmarshal cve data for key %v", it.Key().Data())
		}

		if cve.GetSuppressed() {
			cves[string(id)] = struct{}{}
		}
	}
	return cves, nil
}
