package buildtime

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getPolicy(defaultPolicies []*v1.Policy, name string, t *testing.T) *v1.Policy {
	for _, policy := range defaultPolicies {
		if policy.GetName() == name {
			return policy
		}
	}
	t.Fatalf("Policy %s not found", name)
	return nil
}

func TestDetector(t *testing.T) {
	policySet := image.NewPolicySet(mocks.NewMockDataStore(gomock.NewController(t)))
	detector := NewDetector(policySet)
	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies()
	require.NoError(t, err)

	require.NoError(t, policySet.UpsertPolicy(getPolicy(defaultPolicies, "Latest tag", t)))

	for _, testCase := range []struct {
		image          *storage.Image
		expectedAlerts int
	}{
		{
			image:          &storage.Image{Name: &storage.ImageName{Tag: "latest"}},
			expectedAlerts: 1,
		},
		{
			image:          &storage.Image{Id: "AAA", Name: &storage.ImageName{Tag: "latest"}},
			expectedAlerts: 1,
		},
		{
			image:          &storage.Image{Id: "AAA", Name: &storage.ImageName{Tag: "OLDEST"}},
			expectedAlerts: 0,
		},
	} {
		t.Run(proto.MarshalTextString(testCase.image), func(t *testing.T) {
			alerts, err := detector.Detect(testCase.image)
			require.NoError(t, err)
			assert.Len(t, alerts, testCase.expectedAlerts)
		})

	}
}
