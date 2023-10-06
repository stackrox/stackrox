package m190tom191

import (
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func migrate(_ *types.Databases) error {
	// broken.  turning off to unblock tests.  fixing under ROX-20017
	log.Infof("Skipping migration of plop records")

	return nil
}
