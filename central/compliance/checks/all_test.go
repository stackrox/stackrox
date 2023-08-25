package checks

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stretchr/testify/assert"
)

func TestAllCheckIDsAreValid(t *testing.T) {
	allChecks := framework.RegistrySingleton().GetAll()
	var allControls []string

	registryInstance, err := standards.NewRegistry(framework.RegistrySingleton(), metadata.AllStandards...)
	assert.NoError(t, err)
	for _, standard := range registryInstance.AllStandards() {
		for _, ctrl := range standard.AllControls() {
			allControls = append(allControls, ctrl.QualifiedID())
		}
	}

	for _, check := range allChecks {
		assert.Containsf(t, allControls, check.ID(), "Check %s does not correspond to a control in any compliance standard", check.ID())
	}
}
