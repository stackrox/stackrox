package dbs

import (
	"context"
	"io"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/badgerutils"
	"github.com/stackrox/rox/pkg/binenc"
)

const (
	backupVersion uint32 = 2
)

// NewBadgerBackup returns a generator of BadgerDB backups.
func NewBadgerBackup(db *badger.DB) *BadgerBackup {
	return &BadgerBackup{db: db}
}

// BadgerBackup is an implementation of a StreamGenerator which writes a backup of BadgerDB to the input io.Writer.
type BadgerBackup struct {
	db *badger.DB
}

// WriteTo writes a backup of BadgerDB to the input io.Writer.
func (bgen *BadgerBackup) WriteTo(ctx context.Context, out io.Writer) error {
	// Write backup version out to writer as first 4 bytes
	magic := binenc.BigEndian.EncodeUint32(badgerutils.MagicNumber)
	if _, err := out.Write(magic); err != nil {
		return errors.Wrap(err, "error writing magic to output")
	}

	version := binenc.BigEndian.EncodeUint32(backupVersion)
	if _, err := out.Write(version); err != nil {
		return errors.Wrap(err, "error writing version to output")
	}

	stream := bgen.db.NewStream()
	stream.NumGo = 8

	_, err := stream.LegacyBackup(out, 0)
	if err != nil {
		return errors.Wrap(err, "could not create badger backup")
	}
	return nil
}
