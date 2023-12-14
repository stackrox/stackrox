package m195tom196

import (
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

//nolint:revive
func migrate(database *types.Databases) error {
	// This migration has been reverted due to the feature being disabled by default.
	// We can't easily revert due to the way migrations stack on top of each other.
	// The original changes can be found in commit 7e917d4139d04679efa4bbf14e389f697fb67467
	// Or via https://github.com/stackrox/stackrox/tree/7e917d4139d04679efa4bbf14e389f697fb67467/migrator/migrations/m_195_to_m_196_vuln_request_users
	log.Debugf("Skipping migration 195 to 196")
	return nil
}
