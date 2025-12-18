package postgresv1

import (
	"io"
	"math"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/restore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

var (
	log = logging.LoggerForModule()
)

func restorePostgresDB(_ common.RestoreFileContext, fileReader io.Reader, _ int64) error {
	log.Debug("restorePostgresDB")
	err := restore.LoadRestoreStream(fileReader)
	if err != nil {
		return errors.Wrap(err, "unable to restore postgres")
	}

	return nil
}

func checkPostgresSize(_ common.RestoreFileContext, fileReader io.Reader, size int64) error {
	// When using managed services, Postgres space is not a concern at this time.
	if env.ManagedCentral.BooleanSetting() {
		return nil
	}

	bytes := make([]byte, size)

	bytesRead, err := fileReader.Read(bytes)
	if int64(bytesRead) < size || (err != nil && err != io.EOF) {
		log.Warnf("Could not determine free disk space for Postgres: %v. Assuming free space is sufficient.", err)
		return nil
	}

	restoreBytes, err := strconv.ParseInt(string(bytes), 10, 64)
	if err != nil {
		log.Warnf("Could not determine free disk space for Postgres: %v. Assuming free space is sufficient.", err)
		return nil
	}

	requiredBytes := int64(math.Ceil(float64(restoreBytes) * (1.0 + migrations.CapacityMarginFraction)))

	_, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return errors.Wrap(err, "Could not parse postgres config")
	}

	availableBytes, err := pgadmin.GetRemainingCapacity(dbConfig)
	if err != nil {
		log.Warnf("Could not determine free disk space for Postgres: %v. Assuming free space is sufficient for %d bytes.", err, requiredBytes)
		return nil
	}

	hasSpace := float64(availableBytes) > float64(requiredBytes)
	if !hasSpace {
		return errors.Errorf("restoring backup requires %d bytes of free disk space, but Postgres only has %d bytes available", requiredBytes, availableBytes)
	}

	return nil
}

func checkMigrationVersion(_ common.RestoreFileContext, fileReader io.Reader, size int64) error {
	bytes := make([]byte, size)

	bytesRead, err := fileReader.Read(bytes)
	if int64(bytesRead) < size || (err != nil && err != io.EOF) {
		log.Errorf("Could not determine RHACS version for Postgres: %v. Assuming invalid version.", err)
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	var version migrations.MigrationVersion
	err = yaml.Unmarshal(bytes, &version)
	if err != nil {
		log.Errorf("Could not parse RHACS version: %v. Assuming invalid version.", err)
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	if version.SeqNum < migrations.MinimumSupportedDBVersionSeqNum() {
		// Use FailedPrecondition status to indicate this is a permanent validation failure,
		// not a transient error that should be retried.
		errMsg := errors.Errorf("Restoring from this version %q is no longer supported, sequence number %d matching software version %s", version.MainVersion, version.SeqNum, migrations.MinimumSupportedDBVersion())
		log.Error(errMsg)
		return status.Error(codes.FailedPrecondition, errMsg.Error())
	}

	return nil
}
