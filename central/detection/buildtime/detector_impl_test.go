package buildtime

import (
	"testing"

	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/protocompat"
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
	policyToTest := getPolicy(defaultPolicies, "Latest tag", t).CloneVT()
	policyToTest.SetLifecycleStages(append(policyToTest.GetLifecycleStages(), storage.LifecycleStage_BUILD))

	require.NoError(t, policySet.UpsertPolicy(policyToTest))

	for _, testCase := range []struct {
		image                    *storage.Image
		allowedCategories        []string
		expectedAlerts           int
		expectedUnusedCategories []string
	}{
		{
			image:          storage.Image_builder{Name: storage.ImageName_builder{Tag: "latest"}.Build()}.Build(),
			expectedAlerts: 1,
		},
		{
			image:          storage.Image_builder{Id: "AAA", Name: storage.ImageName_builder{Tag: "latest"}.Build()}.Build(),
			expectedAlerts: 1,
		},
		{
			image:          storage.Image_builder{Id: "AAA", Name: storage.ImageName_builder{Tag: "OLDEST"}.Build()}.Build(),
			expectedAlerts: 0,
		},
		{
			image:                    storage.Image_builder{Name: storage.ImageName_builder{Tag: "latest"}.Build()}.Build(),
			allowedCategories:        []string{"Not a category"},
			expectedAlerts:           0,
			expectedUnusedCategories: []string{"Not a category"},
		},
		{
			image:             storage.Image_builder{Name: storage.ImageName_builder{Tag: "latest"}.Build()}.Build(),
			allowedCategories: []string{"DevOps Best Practices"},
			expectedAlerts:    1,
		},
	} {
		t.Run(protocompat.MarshalTextString(testCase.image), func(t *testing.T) {
			filter, getUnusedCategories := detection.MakeCategoryFilter(testCase.allowedCategories)
			alerts, err := detector.Detect(testCase.image, filter)
			require.NoError(t, err)
			require.ElementsMatch(t, testCase.expectedUnusedCategories, getUnusedCategories())
			assert.Len(t, alerts, testCase.expectedAlerts)
		})

	}
}
