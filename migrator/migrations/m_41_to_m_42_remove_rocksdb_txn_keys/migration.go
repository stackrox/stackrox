package m41tom42

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

var (
	extraTransactionPrefix = []string{
		"k8sroles",
		"namespaces",
		"processWhitelistResults",
		"processWhitelists2",
		"risk",
		"rolebindings",
		"secrets",
		"service_accounts",
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()

	maxDeleteBatch = 1000
)

func removePrefix(db *gorocksdb.DB, prefix string) error {
	fullPrefix := []byte(fmt.Sprintf("transactions%s\x00", prefix))

	wb := gorocksdb.NewWriteBatch()

	it := db.NewIterator(readOpts)
	defer it.Close()

	var totalRemoved int
	for it.Seek(fullPrefix); it.ValidForPrefix(fullPrefix); it.Next() {
		wb.Delete(it.Key().Data())
		totalRemoved++
		if wb.Count() == maxDeleteBatch {
			if err := db.Write(writeOpts, wb); err != nil {
				return err
			}
			wb.Clear()
		}
	}
	// Writes the remaining
	if err := db.Write(writeOpts, wb); err != nil {
		return err
	}
	log.WriteToStderrf("Removed %d extra transaction keys for %s", totalRemoved, prefix)
	return nil
}

func removeKeys(db *gorocksdb.DB) error {
	if db == nil {
		return nil
	}
	for _, prefix := range extraTransactionPrefix {
		if err := removePrefix(db, prefix); err != nil {
			return errors.Wrapf(err, "error removing transaction prefixes for %s", prefix)
		}
	}
	return nil
}

var (
	migration = types.Migration{
		StartingSeqNum: 41,
		VersionAfter:   storage.Version{SeqNum: 42},
		Run: func(databases *types.Databases) error {
			return removeKeys(databases.RocksDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
