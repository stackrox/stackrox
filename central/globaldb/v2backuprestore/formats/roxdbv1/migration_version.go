package roxdbv1

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
)

func restoreMigrationVersion(ctx common.RestoreFileContext, fileReader io.Reader, _ int64) error {
	versionFile, err := ctx.OpenFile(migrations.MigrationVersionFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return errors.Wrap(err, "could not create migration version file")
	}
	defer utils.IgnoreError(versionFile.Close)

	versionFileName := versionFile.Name()
	if _, err := io.Copy(versionFile, fileReader); err != nil {
		return errors.Wrap(err, "could not write data to version file")
	}
	if err := versionFile.Close(); err != nil {
		return errors.Wrap(err, "could not close version file")
	}

	// Validate version file
	log.Infof("Validate restore version %s", versionFileName)
	ver, err := migrations.Read(ctx.OutputDir())
	if err != nil {
		return err
	}
	if ver.SeqNum > migrations.CurrentDBVersionSeqNum() || version.CompareVersions(ver.MainVersion, version.GetMainVersion()) > 0 {
		return errors.Errorf("Cannot restore databases from higher version %+v, expect version <= %s and sequence number <= %d", *ver, version.GetMainVersion(), ver.SeqNum)
	}
	return nil
}
