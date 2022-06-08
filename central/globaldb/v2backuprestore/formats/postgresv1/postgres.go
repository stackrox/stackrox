package postgresv1

import (
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/restore"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	scratchPath = "postgresScratch"
)

var (
	log = logging.LoggerForModule()
)

func restorePostgresDB(ctx common.RestoreFileContext, fileReader io.Reader, size int64) error {

	err := restore.LoadRestoreStream(fileReader)
	if err != nil {
		return errors.Wrap(err, "unable to restore postgres")
	}

	return nil

	//tmpDir, err := common.FindTmpPath(size, scratchPath)
	//if err != nil {
	//	return err
	//}
	//defer utils.IgnoreError(func() error { return os.RemoveAll(tmpDir) })
	//
	//// Dump the contents of the tar to the tmpDir
	//err = pkgTar.ToPath(tmpDir, fileReader)
	//if err != nil {
	//	return errors.Wrap(err, "unable to untar postgres backup to scratch path")
	//}
	//
	//err = restore.LoadRestore(tmpDir)
	//if err != nil {
	//	return errors.Wrap(err, "unable to restore postgres")
	//}
	//
	//return os.RemoveAll(tmpDir)
}
