package m213tom214

import (
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func migrate(_ *types.Databases) error {
	// This migration has been reverted due to the feature being disabled by default.
	// We can't easily revert due to the way migrations stack on top of each other.
	// The original changes can be found in commit db2652bb58a054211f32eed4ac18abfe17074ea0
	// Or via https://github.com/stackrox/stackrox/commit/db2652bb58a054211f32eed4ac18abfe17074ea0
	log.Debugf("Skipping migration 213 to 214")
	return nil
}
