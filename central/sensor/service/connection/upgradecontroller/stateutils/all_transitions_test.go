package stateutils

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stretchr/testify/assert"
)

func TestAllTransitionsAreWellFormed(t *testing.T) {
	for _, transition := range allTransitions {
		t.Run(fmt.Sprintf("transition_%+v", transition), func(t *testing.T) {
			if transition.noStateChange {
				assert.Nil(t, transition.nextState)
			} else {
				assert.NotNil(t, transition.nextState)
			}
			assert.Contains(t, sensorupgrader.Workflows(), transition.workflowToExecute)
		})
	}
}
