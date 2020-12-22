package export

import (
	"archive/zip"
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/cas"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/backup/generators/dbs"
	"github.com/stackrox/rox/pkg/backup"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

// Backup backs up the given databases (optionally removing secrets) and writes a ZIP archive to the given writer.
func Backup(ctx context.Context, boltDB *bolt.DB, rocksDB *rocksdb.RocksDB, includeCerts bool, out io.Writer) error {
	zipWriter := zip.NewWriter(out)
	defer utils.IgnoreError(zipWriter.Close)

	if err := generators.PutStreamInZip(dbs.NewBoltBackup(boltDB), backup.BoltFileName).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up bolt")
	}

	if err := generators.PutTarInZip(generators.PutDirectoryInTar(dbs.NewRocksBackup(rocksDB)), backup.RocksFileName).WriteTo(ctx, zipWriter); err != nil {
		return errors.Wrap(err, "backing up badger")
	}

	if includeCerts {
		if err := generators.PutPathMapInZip(cas.NewCertsBackup(), backup.KeysBaseFolder).WriteTo(ctx, zipWriter); err != nil {
			return errors.Wrap(err, "backing up certificates")
		}
	}

	return zipWriter.Close()
}
