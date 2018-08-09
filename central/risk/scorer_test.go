package risk

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/dnrintegration"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/api/v1"
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
					Severity: v1.Severity_CRITICAL_SEVERITY,
				},
			},
		},
	}, &getters.MockDNRIntegrationGetter{
		MockDNRIntegration: &getters.MockDNRIntegration{
			ExpectedNamespace:   "",
			ExpectedServiceName: "",
			MockAlerts: []dnrintegration.PolicyAlert{
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy1", SeverityWord: "MEDIUM", SeverityScore: 50},
			},
			MockError: nil,
		},
		Exists: true,
	})

	// Without user defined function
	expectedRiskScore := 6.048
	expectedRiskResults := []*v1.Risk_Result{
		{
			Name: multipliers.DnrAlertsHeading,
			Factors: []string{
				"FakePolicy0 (Severity: CRITICAL)",
				"FakePolicy1 (Severity: MEDIUM)",
			},
			Score: 1.5,
		},
		{
			Name:    multipliers.PolicyViolationsHeading,
			Factors: []string{"Test (severity: Critical)"},
			Score:   1.2,
		},
		{
			Name: multipliers.VulnsHeading,
			Factors: []string{
				"Image contains 2 CVEs with CVSS scores ranging between 5.0 and 5.0",
			},
			Score: 1.05,
		},
		{
			Name: multipliers.ServiceConfigHeading,
			Factors: []string{
				"Volumes rw volume were mounted RW",
				"Secrets secret are used inside the deployment",
				"Capabilities ALL were added",
				"No capabilities were dropped",
				"A container in the deployment is privileged",
			},
			Score: 2.0,
		},
		{
			Name: multipliers.ReachabilityHeading,
			Factors: []string{
				"Container library/nginx exposes port 8082 to external clients",
				"Container library/nginx exposes port 8083 in the cluster",
				"Container library/nginx exposes port 8084 on node interfaces",
			},
			Score: 1.6,
		},
	}
	actualRisk := scorer.Score(deployment)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	// With user defined function
	for val := 1; val <= 3; val++ {
		mult := &v1.Multiplier{
			Id:   fmt.Sprintf("%d", val),
			Name: fmt.Sprintf("Cluster multiplier %d", val),
			Scope: &v1.Scope{
				Cluster: "cluster",
			},
			Value: float32(val),
		}
		scorer.UpdateUserDefinedMultiplier(mult)
	}

	expectedRiskScore = 36.288
	expectedRiskResults = append(expectedRiskResults, []*v1.Risk_Result{
		{
			Name: "Cluster multiplier 3",
			Factors: []string{
				"Deployment matched scope 'cluster:cluster'",
			},
			Score: 3.0,
		},
		{
			Name: "Cluster multiplier 2",
			Factors: []string{
				"Deployment matched scope 'cluster:cluster'",
			},
			Score: 2.0,
		},
		{
			Name: "Cluster multiplier 1",
			Factors: []string{
				"Deployment matched scope 'cluster:cluster'",
			},
			Score: 1.0,
		},
	}...)
	actualRisk = scorer.Score(deployment)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)
}

func getMockDeployment() *v1.Deployment {
	return &v1.Deployment{
		ClusterId: "cluster",
		Containers: []*v1.Container{
			{
				Volumes: []*v1.Volume{
					{
						Name:     "readonly",
						ReadOnly: true,
					},
				},
				Secrets: []*v1.EmbeddedSecret{
					{
						Name: "secret",
					},
				},
				SecurityContext: &v1.SecurityContext{
					AddCapabilities: []string{
						"ALL",
					},
					Privileged: true,
				},
				Image: &v1.Image{
					Name: &v1.ImageName{
						FullName: "docker.io/library/nginx:1.10",
						Registry: "docker.io",
						Remote:   "library/nginx",
						Tag:      "1.10",
					},
					Scan: &v1.ImageScan{
						Components: []*v1.ImageScanComponent{
							{
								Vulns: []*v1.Vulnerability{
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
				Ports: []*v1.PortConfig{
					{
						Name:          "Port1",
						ContainerPort: 22,
						Exposure:      v1.PortConfig_EXTERNAL,
						ExposedPort:   8082,
					},
					{
						Name:          "Port2",
						ContainerPort: 23,
						Exposure:      v1.PortConfig_INTERNAL,
						ExposedPort:   8083,
					},
					{
						Name:          "Port3",
						ContainerPort: 24,
						Exposure:      v1.PortConfig_NODE,
						ExposedPort:   8084,
					},
				},
			},
			{
				Volumes: []*v1.Volume{
					{
						Name: "rw volume",
					},
				},
				SecurityContext: &v1.SecurityContext{},
			},
		},
	}
}
