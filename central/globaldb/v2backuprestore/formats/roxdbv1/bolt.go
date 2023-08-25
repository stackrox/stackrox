//go:build amd64

package roxdbv1

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/odirect"
	"github.com/stackrox/rox/pkg/utils"
	"go.etcd.io/bbolt"
)

func restoreBoltDB(ctx common.RestoreFileContext, fileReader io.Reader, _ int64) error {
	boltFile, err := ctx.OpenFile(bolthelper.DBFileName, os.O_CREATE|os.O_RDWR|odirect.GetODirectFlag(), 0600)
	if err != nil {
		return errors.Wrap(err, "could not create bolt file")
	}
	defer utils.IgnoreError(boltFile.Close)

	boltFileName := boltFile.Name()

	if _, err := io.Copy(boltFile, fileReader); err != nil {
		return errors.Wrap(err, "could not write data to bolt file")
	}
	if err := boltFile.Close(); err != nil {
		return errors.Wrap(err, "could not close bolt file")
	}

	ctx.CheckAsync(func(_ common.RestoreProcessContext) error { return validateBoltDB(boltFileName) })
	return nil
}

func validateBoltDB(boltFilePath string) error {
	opts := *bbolt.DefaultOptions
	opts.ReadOnly = true
	db, err := bbolt.Open(boltFilePath, 0600, &opts)
	if err != nil {
		return errors.Wrap(err, "could not open bolt database")
	}
	if err := db.Close(); err != nil {
		return errors.Wrap(err, "could not close bolt database after opening")
	}
	return nil
}
