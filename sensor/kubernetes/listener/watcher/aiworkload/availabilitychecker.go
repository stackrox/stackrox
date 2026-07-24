package aiworkload

import (
	"github.com/stackrox/rox/pkg/aiworkload"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

func NewAvailabilityChecker() availability.Checker {
	return availability.NewChecker(aiworkload.GetInferenceServiceGV(), aiworkload.GetInferenceServiceResources())
}
