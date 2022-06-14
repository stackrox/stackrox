package cleanup

import (
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

func cleanupState(ctx *upgradectx.UpgradeContext, own bool) error {
	c := &cleaner{ctx: ctx}
	return c.CleanupState(own)
}

// OwnState cleans up state resources belonging to this upgrader process.
func OwnState(ctx *upgradectx.UpgradeContext) error {
	return cleanupState(ctx, true)
}

// ForeignState cleans up state resources belonging to other upgrader processes.
func ForeignState(ctx *upgradectx.UpgradeContext) error {
	return cleanupState(ctx, false)
}

// Owner cleans up the owning deployment.
func Owner(ctx *upgradectx.UpgradeContext) error {
	c := &cleaner{ctx: ctx}
	return c.CleanupOwner()
}
