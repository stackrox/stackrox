package risk

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestScore(t *testing.T) {
	deployment := getMockDeployment()
	scorer := NewScorer(&getters.MockAlertsGetter{
		Alerts: []*v1.ListAlert{
			{
				Deployment: &v1.ListAlertDeployment{},
				Policy: &v1.ListAlertPolicy{
					Name:     "Test",
					Severity: storage.Severity_CRITICAL_SEVERITY,
				},
			},
		},
	})

	// Without user defined function
	expectedRiskScore := 7.2128
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name:    multipliers.PolicyViolationsHeading,
			Factors: []*storage.Risk_Result_Factor{{Message: "Test (severity: Critical)"}},
			Score:   1.96,
		},
		{
			Name: multipliers.VulnsHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image contains 2 CVEs with CVSS scores ranging between 5.0 and 5.0"},
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
				{Message: "Container library/nginx exposes port 8082 to external clients"},
				{Message: "Container library/nginx exposes port 8083 in the cluster"},
				{Message: "Container library/nginx exposes port 8084 on node interfaces"},
			},
			Score: 1.6,
		},
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

	expectedRiskScore = 43.2768
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
	actualRisk = scorer.Score(deployment)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)
}

func getMockDeployment() *storage.Deployment {
	return &storage.Deployment{
		ClusterId: "cluster",
		Containers: []*storage.Container{
			{
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
				},
				Ports: []*storage.PortConfig{
					{
						Name:          "Port1",
						ContainerPort: 22,
						Exposure:      storage.PortConfig_EXTERNAL,
						ExposedPort:   8082,
					},
					{
						Name:          "Port2",
						ContainerPort: 23,
						Exposure:      storage.PortConfig_INTERNAL,
						ExposedPort:   8083,
					},
					{
						Name:          "Port3",
						ContainerPort: 24,
						Exposure:      storage.PortConfig_NODE,
						ExposedPort:   8084,
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
