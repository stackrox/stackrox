package buildtime

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	policySet := detection.NewPolicySet(mocks.NewMockDataStore(controller))
	detector := NewDetector(policySet)

	defaultPolicies, err := policies.DefaultPolicies()
	require.NoError(t, err)

	// Load the latest tag policy since that has image fields, and add the BUILD lifecycle so it gets compiled for the
	// buildtime policy set.
	policyToTest := getPolicy(defaultPolicies, "Latest tag", t).Clone()
	policyToTest.LifecycleStages = append(policyToTest.LifecycleStages, storage.LifecycleStage_BUILD)

	require.NoError(t, policySet.UpsertPolicy(policyToTest))

	for _, testCase := range []struct {
		image                    *storage.Image
		allowedCategories        []string
		expectedAlerts           int
		expectedUnusedCategories []string
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
		{
			image:                    &storage.Image{Name: &storage.ImageName{Tag: "latest"}},
			allowedCategories:        []string{"Not a category"},
			expectedAlerts:           0,
			expectedUnusedCategories: []string{"Not a category"},
		},
		{
			image:             &storage.Image{Name: &storage.ImageName{Tag: "latest"}},
			allowedCategories: []string{"DevOps Best Practices"},
			expectedAlerts:    1,
		},
	} {
		t.Run(proto.MarshalTextString(testCase.image), func(t *testing.T) {
			filter, getUnusedCategories := detection.MakeCategoryFilter(testCase.allowedCategories)
			alerts, err := detector.Detect(testCase.image, filter)
			require.NoError(t, err)
			require.ElementsMatch(t, testCase.expectedUnusedCategories, getUnusedCategories())
			assert.Len(t, alerts, testCase.expectedAlerts)
		})

	}
}
