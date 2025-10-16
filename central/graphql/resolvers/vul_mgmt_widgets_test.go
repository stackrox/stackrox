package resolvers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestSortBySeverity(t *testing.T) {
	lad := &storage.ListAlertDeployment{}
	lad.SetId("1")
	lad2 := &storage.ListAlertDeployment{}
	lad2.SetId("2")
	lad3 := &storage.ListAlertDeployment{}
	lad3.SetId("3")
	deployments := []*DeploymentsWithMostSevereViolationsResolver{
		{
			deployment: lad,
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
		{
			deployment: lad2,
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      0,
			},
		},
		{
			deployment: lad3,
			policySeverityCounts: &PolicyCounterResolver{
				critical: 2,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
	}

	lad4 := &storage.ListAlertDeployment{}
	lad4.SetId("3")
	lad5 := &storage.ListAlertDeployment{}
	lad5.SetId("1")
	lad6 := &storage.ListAlertDeployment{}
	lad6.SetId("2")
	expected := []*DeploymentsWithMostSevereViolationsResolver{
		{
			deployment: lad4,
			policySeverityCounts: &PolicyCounterResolver{
				critical: 2,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
		{
			deployment: lad5,
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
		{
			deployment: lad6,
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      0,
			},
		},
	}
	sortBySeverity(deployments)
	for i, e := range expected {
		a := deployments[i]
		assert.EqualValues(t, e.policySeverityCounts, a.policySeverityCounts)
		protoassert.Equal(t, e.deployment, a.deployment)
	}
}
