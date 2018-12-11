package multipliers

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Multiplier is the interface that all risk calculations must implement
type Multiplier interface {
	Score(deployment *storage.Deployment) *storage.Risk_Result
}
