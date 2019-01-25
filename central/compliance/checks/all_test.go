package checks

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stretchr/testify/assert"
)

func TestAllCheckIDsAreValid(t *testing.T) {
	allChecks := framework.RegistrySingleton().GetAll()
	var allControls []string
	for _, standard := range standards.RegistrySingleton().AllStandards() {
		allControls = append(allControls, standard.AllControlIDs(true)...)
	}

	for _, check := range allChecks {
		assert.Containsf(t, allControls, check.ID(), "Check %s does not correspond to a control in any compliance standard", check.ID())
	}
}
