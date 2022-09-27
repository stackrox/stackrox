package stateutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// TerminalStates represents terminal states -- once an upgrade is in one of these states,
	// it never gets out.
	TerminalStates = set.NewFrozenSet(
		storage.UpgradeProgress_UPGRADE_COMPLETE,
		storage.UpgradeProgress_PRE_FLIGHT_CHECKS_FAILED,
		storage.UpgradeProgress_UPGRADE_INITIALIZATION_ERROR,
		storage.UpgradeProgress_UPGRADE_TIMED_OUT,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLED_BACK,
		storage.UpgradeProgress_UPGRADE_ERROR_ROLLBACK_FAILED,
		storage.UpgradeProgress_UPGRADE_ERROR_UNKNOWN,
	)
)
