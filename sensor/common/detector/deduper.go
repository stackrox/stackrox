package detector

import (
	"github.com/stackrox/rox/generated/storage"
)

// TODO(ROX-10638): Remove this deduper
// doNothingDeduper evaluates if a run of detection is needed
type doNothingDeduper struct {
}

func newDoNothingDeduper() *doNothingDeduper {
	return &doNothingDeduper{}
}

func (d *doNothingDeduper) reset() {
}

func (d *doNothingDeduper) addDeployment(deployment *storage.Deployment) {}

func (d *doNothingDeduper) needsProcessing(deployment *storage.Deployment) bool {
	return true
}

func (d *doNothingDeduper) removeDeployment(id string) {}
