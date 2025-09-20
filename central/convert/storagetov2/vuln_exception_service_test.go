package storagetov2

import (
	"testing"

	"github.com/stackrox/rox/central/convert/testutils"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
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

	protoassert.Equal(
		t,
		func() *v2.VulnerabilityException {
			req := testutils.GetTestVulnExceptionWithUpdate(t)
			req.GetDeferralRequest().Expiry = &v2.ExceptionExpiry{
				ExpiryType: v2.ExceptionExpiry_TIME,
				ExpiresOn:  nil,
			}
			return req
		}(),
		func() *v2.VulnerabilityException {
			req := testutils.GetTestVulnRequestWithUpdate(t)
			req.GetDeferralReq().Expiry = &storage.RequestExpiry{
				ExpiryType: storage.RequestExpiry_TIME,
				Expiry:     nil,
			}
			return VulnerabilityException(req)
		}(),
	)
}
