package runner

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/sensorupgrader"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowsAreValid(t *testing.T) {
	t.Parallel()

	r := &runner{}
	workflows := sensorupgrader.Workflows()
	stages := r.Stages()

	for workflow, stageIDs := range workflows {
		for _, stageID := range stageIDs {
			_, ok := stages[stageID]
			assert.Truef(t, ok, "workflow %s references missing stage %s", workflow, stageID)
		}
	}
}
