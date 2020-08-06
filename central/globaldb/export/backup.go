package export

import (
	"archive/zip"
	"context"
	"io"

	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/dbs"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/utils"
)

// Backup backs up the given databases (optionally removing secrets) and writes a ZIP archive to the given writer.
func Backup(ctx context.Context, boltDB *bolt.DB, rocksDB *rocksdb.RocksDB, out io.Writer) error {
	zipWriter := zip.NewWriter(out)
	defer utils.IgnoreError(zipWriter.Close)

	if err := generators.PutStreamInZip(dbs.NewBoltBackup(boltDB), boltFileName).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up bolt")
	}

	if err := generators.PutTarInZip(generators.PutDirectoryInTar(dbs.NewRocksBackup(rocksDB)), rocksFileName).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up badger")
	}

	return zipWriter.Close()
}
