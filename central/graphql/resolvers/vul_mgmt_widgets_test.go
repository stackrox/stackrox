package resolvers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestSortBySeverity(t *testing.T) {
	deployments := []*DeploymentsWithMostSevereViolationsResolver{
		{
			deployment: &storage.ListAlertDeployment{
				Id: "1",
			},
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
		{
			deployment: &storage.ListAlertDeployment{
				Id: "2",
			},
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      0,
			},
		},
		{
			deployment: &storage.ListAlertDeployment{
				Id: "3",
			},
			policySeverityCounts: &PolicyCounterResolver{
				critical: 2,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
	}

	expected := []*DeploymentsWithMostSevereViolationsResolver{
		{
			deployment: &storage.ListAlertDeployment{
				Id: "3",
			},
			policySeverityCounts: &PolicyCounterResolver{
				critical: 2,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
		{
			deployment: &storage.ListAlertDeployment{
				Id: "1",
			},
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      1,
			},
		},
		{
			deployment: &storage.ListAlertDeployment{
				Id: "2",
			},
			policySeverityCounts: &PolicyCounterResolver{
				critical: 1,
				high:     2,
				medium:   0,
				low:      0,
			},
		},
	}
	sortBySeverity(deployments)
	assert.EqualValues(t, deployments, expected)
}
