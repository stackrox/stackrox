package v2tostorage

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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

	id := mockIdentity.NewMockIdentity(gomock.NewController(t))
	id.EXPECT().UID().Return("userID").AnyTimes()
	id.EXPECT().FullName().Return("userName").AnyTimes()
	id.EXPECT().FriendlyName().Return("userName").AnyTimes()
	ctx := authn.ContextWithIdentity(sac.WithAllAccess(context.Background()), id, t)

	assert.EqualValues(
		t,
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnDeferralRequestFull(t)
			// Reset the fields that are nondeterministic.
			req.Id = ""
			req.Name = ""
			req.Approvers = nil
			req.Comments[0].Id = ""
			req.Comments[0].CreatedAt = nil
			return req
		}(),
		func() *storage.VulnerabilityRequest {
			converted := DeferVulnerabilityRequest(ctx,
				testutils.GetTestCreateDeferVulnExceptionRequest(t))
			assert.NotNil(t, converted.GetCreatedAt())
			assert.NotNil(t, converted.GetLastUpdated())
			// Reset the fields that are nondeterministic.
			converted.CreatedAt = nil
			converted.LastUpdated = nil
			converted.Comments[0].Id = ""
			converted.Comments[0].CreatedAt = nil
			return converted
		}(),
	)

	assert.EqualValues(
		t,
		func() *storage.VulnerabilityRequest {
			req := testutils.GetTestVulnFPRequestFull(t)
			// Reset the fields that are nondeterministic.
			req.Id = ""
			req.Name = ""
			req.Approvers = nil
			req.Comments[0].Id = ""
			req.Comments[0].CreatedAt = nil
			return req
		}(),
		func() *storage.VulnerabilityRequest {
			converted := FalsePositiveVulnerabilityRequest(ctx,
				testutils.GetTestCreateFPVulnExceptionRequest(t))
			assert.NotNil(t, converted.GetCreatedAt())
			assert.NotNil(t, converted.GetLastUpdated())
			// Reset the fields that are nondeterministic.
			converted.CreatedAt = nil
			converted.LastUpdated = nil
			converted.Comments[0].Id = ""
			converted.Comments[0].CreatedAt = nil
			return converted
		}(),
	)
}
