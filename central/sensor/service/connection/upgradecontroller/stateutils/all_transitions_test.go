package stateutils

import (
	"fmt"
	"testing"

	"github.com/stackrox/stackrox/pkg/sensorupgrader"
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

func TestAllTerminalStatesResultInCleanUp(t *testing.T) {
	for _, transition := range allTransitions {
		t.Run(fmt.Sprintf("transition_%+v", transition), func(t *testing.T) {
			if transition.nextState == nil {
				return
			}
			if TerminalStates.Contains(*transition.nextState) {
				assert.Equal(t, sensorupgrader.CleanupWorkflow, transition.workflowToExecute)
			}
		})
	}
}
