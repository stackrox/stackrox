package risk

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

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
				Secrets: []*v1.Secret{
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
					},
					{
						Name:          "Port2",
						ContainerPort: 23,
						Exposure:      v1.PortConfig_INTERNAL,
					},
					{
						Name:          "Port3",
						ContainerPort: 8080,
						Exposure:      v1.PortConfig_NODE,
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

func TestScore(t *testing.T) {
	deployment := getMockDeployment()
	scorer := NewScorer(&mockAlertsGetter{
		alerts: []*v1.Alert{
			{
				Deployment: deployment,
				Policy: &v1.Policy{
					Name:     "Test",
					Severity: v1.Severity_CRITICAL_SEVERITY,
				},
			},
		},
	}, &mockDNRIntegrationGetter{})

	// Without user defined function
	expectedRisk := &v1.Risk{
		Score: 4.032,
		Results: []*v1.Risk_Result{
			{
				Name: serviceConfigHeading,
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
				Name: reachabilityHeading,
				Factors: []string{
					"Container library/nginx exposes port 22 to external clients",
					"Container library/nginx exposes port 23 in the cluster",
					"Container library/nginx exposes port 8080 on node interfaces",
				},
				Score: 1.6,
			},
			{
				Name:    policyViolationsHeading,
				Factors: []string{"Test (severity: Critical)"},
				Score:   1.2,
			},
			{
				Name: vulnsHeading,
				Factors: []string{
					"Image contains 2 CVEs with CVSS scores ranging between 5.0 and 5.0",
				},
				Score: 1.05,
			},
		},
	}
	actualRisk := scorer.Score(deployment)
	assert.Equal(t, expectedRisk, actualRisk)

	// With user defined function
	mult := &v1.Multiplier{
		Name: "Cluster multiplier",
		Scope: &v1.Scope{
			Cluster: "cluster",
		},
		Value: 2.0,
	}
	scorer.UpdateUserDefinedMultiplier(mult)
	expectedRisk = &v1.Risk{
		Score: 8.064,
		Results: []*v1.Risk_Result{
			{
				Name: serviceConfigHeading,
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
				Name: "Cluster multiplier",
				Factors: []string{
					"Deployment matched scope 'cluster:cluster'",
				},
				Score: 2.0,
			},
			{
				Name: reachabilityHeading,
				Factors: []string{
					"Container library/nginx exposes port 22 to external clients",
					"Container library/nginx exposes port 23 in the cluster",
					"Container library/nginx exposes port 8080 on node interfaces",
				},
				Score: 1.6,
			},
			{
				Name:    policyViolationsHeading,
				Factors: []string{"Test (severity: Critical)"},
				Score:   1.2,
			},
			{
				Name: vulnsHeading,
				Factors: []string{
					"Image contains 2 CVEs with CVSS scores ranging between 5.0 and 5.0",
				},
				Score: 1.05,
			},
		},
	}
	actualRisk = scorer.Score(deployment)
	assert.Equal(t, expectedRisk, actualRisk)
}
