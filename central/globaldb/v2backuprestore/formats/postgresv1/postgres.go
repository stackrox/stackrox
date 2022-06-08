package postgresv1

import (
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/restore"
)

func restorePostgresDB(ctx common.RestoreFileContext, fileReader io.Reader, size int64) error {

	err := restore.LoadRestoreStream(fileReader)
	if err != nil {
		return errors.Wrap(err, "unable to restore postgres")
	}

	return nil
}
