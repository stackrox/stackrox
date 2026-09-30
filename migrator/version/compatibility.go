package version

import (
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version"
)

// CheckCompatibility checks the selected database before any application changes.
func CheckCompatibility(source *migrations.MigrationVersion) error {
	bypassed, err := migrations.CheckUpgradeCompatibility(*source, version.GetMainVersion(),
		migrations.CurrentDBVersionSeqNum(), env.UnsafeAllowUnsupportedUpgrade.BooleanSetting())
	if bypassed != "" {
		log.WriteToStderrf("WARNING: ROX_UNSAFE_ALLOW_UNSUPPORTED_UPGRADE bypassed compatibility protection: %s Database sequence: %d. Remove the override after support-assisted recovery.", bypassed, source.SeqNum)
	}
	return err
}
