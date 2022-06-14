package handlers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_stateToString(t *testing.T) {
	// if this test case is failing, please add the new enum value to stateToStringMap
	for i := range storage.ComplianceState_name {
		state := storage.ComplianceState(i)
		val := stateToString(state)
		if state != storage.ComplianceState_COMPLIANCE_STATE_UNKNOWN {
			assert.NotEqual(t, "Unknown", val, `Compliance state %q should not return "Unknown"`, state.String())
		}
	}
}
