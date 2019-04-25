package risk

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	bindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	saMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
)

func TestScore(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRoles := roleMocks.NewMockDataStore(mockCtrl)
	mockBindings := bindingMocks.NewMockDataStore(mockCtrl)
	mockServiceAccounts := saMocks.NewMockDataStore(mockCtrl)

	deployment := getMockDeployment()
	scorer := NewScorer(&getters.MockAlertsGetter{
		Alerts: []*storage.ListAlert{
			{
				Deployment: &storage.ListAlertDeployment{},
				Policy: &storage.ListAlertPolicy{
					Name:     "Test",
					Severity: storage.Severity_CRITICAL_SEVERITY,
				},
			},
		},
	}, mockRoles, mockBindings, mockServiceAccounts)

	// Without user defined function
	expectedRiskScore := 9.016
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name:    multipliers.PolicyViolationsHeading,
			Factors: []*storage.Risk_Result_Factor{{Message: "Test (severity: Critical)"}},
			Score:   1.96,
		},
		{
			Name: multipliers.VulnsHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image \"docker.io/library/nginx:1.10\" (container \"nginx\") contains 2 CVEs with CVSS scores ranging between 5.0 and 5.0"},
			},
			Score: 1.15,
		},
		{
			Name: multipliers.ServiceConfigHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Volumes rw volume were mounted RW"},
				{Message: "Secrets secret are used inside the deployment"},
				{Message: "Capabilities ALL were added"},
				{Message: "No capabilities were dropped"},
				{Message: "A container in the deployment is privileged"},
			},
			Score: 2.0,
		},
		{
			Name: multipliers.ReachabilityHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Port 22 is exposed to external clients"},
				{Message: "Port 23 is exposed in the cluster"},
				{Message: "Port 24 is exposed on node interfaces"},
			},
			Score: 1.6,
		},
		{
			Name: multipliers.ImageAgeHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Deployment contains an image 180 days old"},
			},
			Score: 1.25,
		},
	}

	if features.K8sRBAC.Enabled() {
		mockServiceAccounts.EXPECT().SearchRawServiceAccounts(gomock.Any()).Return(nil, nil)
	}

	actualRisk := scorer.Score(deployment)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	// With user defined function
	for val := 1; val <= 3; val++ {
		mult := &storage.Multiplier{
			Id:   fmt.Sprintf("%d", val),
			Name: fmt.Sprintf("Cluster multiplier %d", val),
			Scope: &storage.Scope{
				Cluster: "cluster",
			},
			Value: float32(val),
		}
		scorer.UpdateUserDefinedMultiplier(mult)
	}

	expectedRiskScore = 54.096
	expectedRiskResults = append(expectedRiskResults, []*storage.Risk_Result{
		{
			Name: "Cluster multiplier 3",
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Deployment matched scope 'cluster:cluster'"},
			},
			Score: 3.0,
		},
		{
			Name: "Cluster multiplier 2",
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Deployment matched scope 'cluster:cluster'"},
			},
			Score: 2.0,
		},
		{
			Name: "Cluster multiplier 1",
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Deployment matched scope 'cluster:cluster'"},
			},
			Score: 1.0,
		},
	}...)

	if features.K8sRBAC.Enabled() {
		mockServiceAccounts.EXPECT().SearchRawServiceAccounts(gomock.Any()).Return(nil, nil)
	}

	actualRisk = scorer.Score(deployment)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}

func getMockDeployment() *storage.Deployment {
	return &storage.Deployment{
		ClusterId: "cluster",
		Ports: []*storage.PortConfig{
			{
				Name:          "Port1",
				ContainerPort: 22,
				Exposure:      storage.PortConfig_EXTERNAL,
			},
			{
				Name:          "Port2",
				ContainerPort: 23,
				Exposure:      storage.PortConfig_INTERNAL,
			},
			{
				Name:          "Port3",
				ContainerPort: 24,
				Exposure:      storage.PortConfig_NODE,
			},
		},
		Containers: []*storage.Container{
			{
				Name: "nginx",
				Volumes: []*storage.Volume{
					{
						Name:     "readonly",
						ReadOnly: true,
					},
				},
				Secrets: []*storage.EmbeddedSecret{
					{
						Name: "secret",
					},
				},
				SecurityContext: &storage.SecurityContext{
					AddCapabilities: []string{
						"ALL",
					},
					Privileged: true,
				},
				Image: &storage.Image{
					Name: &storage.ImageName{
						FullName: "docker.io/library/nginx:1.10",
						Registry: "docker.io",
						Remote:   "library/nginx",
						Tag:      "1.10",
					},
					Scan: &storage.ImageScan{
						Components: []*storage.ImageScanComponent{
							{
								Vulns: []*storage.Vulnerability{
									{
										Cvss: 5,
									},
									{
										Cvss: 5,
									},
								},
							},
						},
					},
					Metadata: &storage.ImageMetadata{
						V1: &storage.V1Metadata{
							Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(180 * 24 * time.Hour))),
						},
					},
				},
			},
			{
				Volumes: []*storage.Volume{
					{
						Name: "rw volume",
					},
				},
				SecurityContext: &storage.SecurityContext{},
			},
		},
	}
}
