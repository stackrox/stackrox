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
					{Name: "readonly",
						ReadOnly: true,
					},
					{
						Name: "secret",
						Type: "secret",
					},
				},
				SecurityContext: &v1.SecurityContext{
					AddCapabilities: []string{
						"ALL",
					},
					Privileged: true,
				},
				Image: &v1.Image{
					Scan: &v1.ImageScan{
						Components: []*v1.ImageScanComponents{
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
	scorer := NewScorer()

	deployment := getMockDeployment()
	// Without user defined function
	expectedRisk := &v1.Risk{
		Score: 2.1,
		Results: []*v1.Risk_Result{
			{
				Name: "Service Configuration Heuristic",
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
				Name: "Vulnerability Heuristic",
				Factors: []string{
					"Normalized and discounted sum of 2 CVSS scores",
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
		Score: 4.2,
		Results: []*v1.Risk_Result{
			{
				Name: "Service Configuration Heuristic",
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
				Name: "Vulnerability Heuristic",
				Factors: []string{
					"Normalized and discounted sum of 2 CVSS scores",
				},
				Score: 1.05,
			},
		},
	}
	actualRisk = scorer.Score(deployment)
	assert.Equal(t, expectedRisk, actualRisk)
}
