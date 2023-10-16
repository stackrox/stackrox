package storagetov2

import (
	"testing"

	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilityRequest(t *testing.T) {
	assert.EqualValues(
		t,
		testutils.GetTestVulnDeferralRequestFull(t),
		VulnerabilityRequest(testutils.GetTestVulnDeferralExceptionFull(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnFPRequestFull(t),
		VulnerabilityRequest(testutils.GetTestVulnFPExceptionFull(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnRequestNoComments(t),
		VulnerabilityRequest(testutils.GetTestVulnExceptionNoComments(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnRequestNoUsers(t),
		VulnerabilityRequest(testutils.GetTestVulnExceptionNoUsers(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnRequestWithUpdate(t),
		VulnerabilityRequest(testutils.GetTestVulnExceptionWithUpdate(t)),
	)
}
