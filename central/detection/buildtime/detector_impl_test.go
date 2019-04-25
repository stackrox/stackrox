package buildtime

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/image/index/mappings"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/central/searchbasedpolicies/fields"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getPolicy(defaultPolicies []*storage.Policy, name string, t *testing.T) *storage.Policy {
	for _, policy := range defaultPolicies {
		if policy.GetName() == name {
			return policy
		}
	}
	t.Fatalf("Policy %s not found", name)
	return nil
}

func TestDetector(t *testing.T) {
	controller := gomock.NewController(t)
	compilerWithoutProcessIndicators := detection.NewPolicyCompiler(matcher.NewBuilder(fields.NewRegistry(nil), mappings.OptionsMap))
	policySet := detection.NewPolicySet(mocks.NewMockDataStore(controller), compilerWithoutProcessIndicators)
	detector := NewDetector(policySet)

	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies()
	require.NoError(t, err)

	// Load the latest tag policy since that has image fields, and add the BUILD lifecycle so it gets compiled for the
	// buildtime policy set.
	policyToTest := protoutils.CloneStoragePolicy(getPolicy(defaultPolicies, "Latest tag", t))
	policyToTest.LifecycleStages = append(policyToTest.LifecycleStages, storage.LifecycleStage_BUILD)

	require.NoError(t, policySet.UpsertPolicy(policyToTest))

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
