package version

import (
	"fmt"

	"github.com/pkg/errors"
	vStore "github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
)

var (
	log = logging.LoggerForModule()
)

// Ensure is an opaque command that ensures that the DB is in a good state by the time it returns.
// It will simply return if the DB version is already what Central expects.
// If the DB is empty, then it will write the current DB version to the DB.
// It will returns an error if the DB is of an old version.
// If Ensure returns an error, the state of the DB is undefined, and it is not safe for Central to try to
// function normally.
func Ensure(versionStore vStore.Store) error {
	version, err := versionStore.GetVersion()
	if err != nil {
		return errors.Wrap(err, "failed to read version from DB")
	}

	if version == nil {
		return errors.New("no DB version found")
	}

	actualSeqNum := int(version.GetSeqNum())
	expectedSeqNum := migrations.CurrentDBVersionSeqNum()

	switch {
	case actualSeqNum < expectedSeqNum:
		return fmt.Errorf("DB version %d below expected version %d", version.GetSeqNum(), migrations.CurrentDBVersionSeqNum())
	case actualSeqNum > expectedSeqNum:
		// This case allows old version centrals to restart for rollbacks and during RollingUpdates
		log.Info("DB version higher than expected version, central can start since it's backward compatible")
		return nil
	default:
		log.Info("Version found in the DB was current. We're good to go!")
		return nil
	}
}
