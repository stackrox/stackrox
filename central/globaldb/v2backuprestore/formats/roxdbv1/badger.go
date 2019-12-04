package roxdbv1

import (
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/badgerutils"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/badgerhelper"
)

func restoreBadger(ctx common.RestoreFileContext, fileReader io.Reader, size int64) error {
	absDirPath, err := ctx.Mkdir(badgerhelper.BadgerDBDirName, 0700)
	if err != nil {
		return errors.Wrap(err, "could not create badger database directory")
	}

	db, err := badgerhelper.New(absDirPath, false)
	if err != nil {
		return errors.Wrapf(err, "could not create new badger DB in empty dir %s", absDirPath)
	}

	if err := badgerutils.Load(fileReader, db); err != nil {
		return errors.Wrap(err, "could not load badger DB backup")
	}

	ctx.CheckAsync(func(_ common.RestoreProcessContext) error {
		return errors.Wrap(db.Close(), "could not close badger DB after loading")
	})

	return nil
}
