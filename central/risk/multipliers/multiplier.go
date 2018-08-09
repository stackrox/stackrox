package multipliers

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// Multiplier is the interface that all risk calculations must implement
type Multiplier interface {
	Score(deployment *v1.Deployment) *v1.Risk_Result
}
