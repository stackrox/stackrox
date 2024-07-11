package storagetov2

import (
	"testing"

	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
)

func TestVulnerabilityException(t *testing.T) {
	protoassert.Equal(
		t,
		testutils.GetTestVulnDeferralExceptionFull(t),
		VulnerabilityException(testutils.GetTestVulnDeferralRequestFull(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnFPExceptionFull(t),
		VulnerabilityException(testutils.GetTestVulnFPRequestFull(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnExceptionNoComments(t),
		VulnerabilityException(testutils.GetTestVulnRequestNoComments(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnExceptionNoUsers(t),
		VulnerabilityException(testutils.GetTestVulnRequestNoUsers(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnExceptionWithUpdate(t),
		VulnerabilityException(testutils.GetTestVulnRequestWithUpdate(t)),
	)
}
