package m51tom52

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
	transactionPrefixes = []string{
		"clusters",
		"clusters_health_status",
		"k8sroles",
		"namespaces",
		"networkentity",
		"networkgraphconfig",
		"processWhitelists2",
		"processWhitelistResults",
		"risk",
		"rolebindings",
		"secrets",
		"service_accounts",
		"integrationhealth",
		"apiTokens",
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()

	maxDeleteBatch = 1000
)

var (
	migration = types.Migration{
		StartingSeqNum: 51,
		VersionAfter:   storage.Version{SeqNum: 52},
		Run: func(databases *types.Databases) error {
			return removeInvalidTransactions(databases.RocksDB)
		},
	}
)

func removeTransactionsForPrefix(db *gorocksdb.DB, prefix string) error {
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

func removeInvalidTransactions(db *gorocksdb.DB) error {
	if db == nil {
		return nil
	}
	for _, prefix := range transactionPrefixes {
		if err := removeTransactionsForPrefix(db, prefix); err != nil {
			return errors.Wrapf(err, "removing transactions for prefix %s", prefix)
		}
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
