package dbs

import (
	"context"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/odirect"
	"github.com/stackrox/rox/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

// NewBoltBackup returns a backup generator for BoltDB backups.
func NewBoltBackup(db *bolt.DB) *BoltBackup {
	return &BoltBackup{db: db}
}

// BoltBackup is an implementation of a StreamGenerator which writes a backup of BoltDB to the input io.Writer.
type BoltBackup struct {
	db *bolt.DB
}

// WriteTo writes a backup of BoltDB to the input io.Writer.
func (bgen *BoltBackup) WriteTo(ctx context.Context, out io.Writer) error {
	tempFile, err := os.CreateTemp("", "bolt-backup-")
	if err != nil {
		return errors.Wrap(err, "could not create temporary file for bolt backup")
	}
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()
	defer utils.IgnoreError(tempFile.Close)

	odirectFlag := odirect.GetODirectFlag()
	err = bgen.db.View(func(tx *bolt.Tx) error {
		tx.WriteFlag = odirectFlag
		_, err := tx.WriteTo(out)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "could not dump bolt database")
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return errors.Wrap(err, "could not rewind to beginning of file")
	}
	dbFileReader := io.ReadCloser(tempFile)
	defer utils.IgnoreError(dbFileReader.Close)

	_, err = io.Copy(out, dbFileReader)
	if err != nil {
		return errors.Wrap(err, "could not copy bolt backup file")
	}
	return nil
}
