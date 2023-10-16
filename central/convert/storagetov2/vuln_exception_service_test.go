package storagetov2

import (
	"testing"

	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilityException(t *testing.T) {
	assert.EqualValues(
		t,
		testutils.GetTestVulnDeferralExceptionFull(t),
		VulnerabilityException(testutils.GetTestVulnDeferralRequestFull(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnFPExceptionFull(t),
		VulnerabilityException(testutils.GetTestVulnFPRequestFull(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnExceptionNoComments(t),
		VulnerabilityException(testutils.GetTestVulnRequestNoComments(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnExceptionNoUsers(t),
		VulnerabilityException(testutils.GetTestVulnRequestNoUsers(t)),
	)

	assert.EqualValues(
		t,
		testutils.GetTestVulnExceptionWithUpdate(t),
		VulnerabilityException(testutils.GetTestVulnRequestWithUpdate(t)),
	)
}
