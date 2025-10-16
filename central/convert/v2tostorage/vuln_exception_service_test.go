package v2tostorage

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/convert/testutils"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestVulnerabilityRequest(t *testing.T) {
	protoassert.Equal(
		t,
		testutils.GetTestVulnDeferralRequestFull(t),
		VulnerabilityRequest(testutils.GetTestVulnDeferralExceptionFull(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnFPRequestFull(t),
		VulnerabilityRequest(testutils.GetTestVulnFPExceptionFull(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnRequestNoComments(t),
		VulnerabilityRequest(testutils.GetTestVulnExceptionNoComments(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnRequestNoUsers(t),
		VulnerabilityRequest(testutils.GetTestVulnExceptionNoUsers(t)),
	)

	protoassert.Equal(
		t,
		testutils.GetTestVulnRequestWithUpdate(t),
		VulnerabilityRequest(testutils.GetTestVulnExceptionWithUpdate(t)),
	)

	protoassert.Equal(
		t,
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnRequestWithUpdate(t)
			re := &storage.RequestExpiry{}
			re.SetExpiryType(storage.RequestExpiry_ALL_CVE_FIXABLE)
			req.GetDeferralReq().SetExpiry(re)
			return req
		}(),
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnExceptionWithUpdate(t)
			ee := &v2.ExceptionExpiry{}
			ee.SetExpiryType(v2.ExceptionExpiry_ALL_CVE_FIXABLE)
			req.GetDeferralRequest().SetExpiry(ee)
			return VulnerabilityRequest(req)
		}(),
	)

	protoassert.Equal(
		t,
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnRequestWithUpdate(t)
			re := &storage.RequestExpiry{}
			re.SetExpiryType(storage.RequestExpiry_ANY_CVE_FIXABLE)
			re.SetExpiresWhenFixed(true)
			req.GetDeferralReq().SetExpiry(re)
			return req
		}(),
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnExceptionWithUpdate(t)
			ee := &v2.ExceptionExpiry{}
			ee.SetExpiryType(v2.ExceptionExpiry_ANY_CVE_FIXABLE)
			req.GetDeferralRequest().SetExpiry(ee)
			return VulnerabilityRequest(req)
		}(),
	)

	protoassert.Equal(
		t,
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnRequestWithUpdate(t)
			re := &storage.RequestExpiry{}
			re.SetExpiryType(storage.RequestExpiry_TIME)
			re.ClearExpiry()
			req.GetDeferralReq().SetExpiry(re)
			return req
		}(),
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnExceptionWithUpdate(t)
			ee := &v2.ExceptionExpiry{}
			ee.SetExpiryType(v2.ExceptionExpiry_TIME)
			ee.ClearExpiresOn()
			req.GetDeferralRequest().SetExpiry(ee)
			return VulnerabilityRequest(req)
		}(),
	)

	id := mockIdentity.NewMockIdentity(gomock.NewController(t))
	id.EXPECT().UID().Return("userID").AnyTimes()
	id.EXPECT().FullName().Return("userName").AnyTimes()
	id.EXPECT().FriendlyName().Return("userName").AnyTimes()
	ctx := authn.ContextWithIdentity(sac.WithAllAccess(context.Background()), id, t)

	protoassert.Equal(
		t,
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnDeferralRequestFull(t)
			// Reset the fields that are nondeterministic.
			req.SetId("")
			req.SetName("")
			req.SetApprovers(nil)
			req.SetApproversV2(nil)
			req.GetComments()[0].SetId("")
			req.GetComments()[0].ClearCreatedAt()
			return req
		}(),
		func() *storage.VulnerabilityRequest {
			converted := DeferVulnerabilityRequest(ctx,
				testutils.GetTestCreateDeferVulnExceptionRequest(t))
			assert.NotNil(t, converted.GetCreatedAt())
			assert.NotNil(t, converted.GetLastUpdated())
			// Reset the fields that are nondeterministic.
			converted.ClearCreatedAt()
			converted.ClearLastUpdated()
			converted.GetComments()[0].SetId("")
			converted.GetComments()[0].ClearCreatedAt()
			return converted
		}(),
	)

	protoassert.Equal(
		t,
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnFPRequestFull(t)
			// Reset the fields that are nondeterministic.
			req.SetId("")
			req.SetName("")
			req.SetApprovers(nil)
			req.SetApproversV2(nil)
			req.GetComments()[0].SetId("")
			req.GetComments()[0].ClearCreatedAt()
			return req
		}(),
		func() *storage.VulnerabilityRequest {
			converted := FalsePositiveVulnerabilityRequest(ctx,
				testutils.GetTestCreateFPVulnExceptionRequest(t))
			assert.NotNil(t, converted.GetCreatedAt())
			assert.NotNil(t, converted.GetLastUpdated())
			// Reset the fields that are nondeterministic.
			converted.ClearCreatedAt()
			converted.ClearLastUpdated()
			converted.GetComments()[0].SetId("")
			converted.GetComments()[0].ClearCreatedAt()
			return converted
		}(),
	)
}
