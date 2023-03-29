package multipliers

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoconv"
)

// GetMockImages returns a slice of mock images
func GetMockImages() []*storage.Image {
	return []*storage.Image{
		{
			Name: &storage.ImageName{
				FullName: "docker.io/library/nginx:1.10",
				Registry: "docker.io",
				Remote:   "library/nginx",
				Tag:      "1.10",
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp1",
						Version: "1.1.1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:      "CVE-2019-0001",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
							},
							{
								Cve:      "CVE-2019-0002",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							},
						},
					},
					{
						Name:    "comp1",
						Version: "1.1.1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:          "CVE-2019-0001",
								Cvss:         5,
								ScoreVersion: storage.EmbeddedVulnerability_V3,
								Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							},
							{
								Cve:      "CVE-2019-0002",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
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
		{
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(90 * 24 * time.Hour))),
				},
			},
		},
	}
}

// GetMockDeployment returns a mock deployment
func GetMockDeployment() *storage.Deployment {
	return &storage.Deployment{
		Name:                         "mock-deployment",
		ServiceAccount:               "service-account",
		ClusterId:                    "cluster",
		Namespace:                    "namespace",
		AutomountServiceAccountToken: true,
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
				Name: "containerName",
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
				Image: types.ToContainerImage(GetMockImages()[0]),
			},
			{
				Name:  "Container2",
				Image: types.ToContainerImage(GetMockImages()[1]),
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

// GetMockNodes returns a slice of mock nodes
func GetMockNodes() []*storage.Node {
	return []*storage.Node{
		{
			Name: "node1",
			Scan: &storage.NodeScan{
				Components: []*storage.EmbeddedNodeScanComponent{
					{
						Name:    "kubelet",
						Version: "1.16.9",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:          "CVE-2020-8558",
								Cvss:         5.4,
								ScoreVersion: storage.EmbeddedVulnerability_V3,
								Severity:     storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
						},
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "CVE-2020-8558",
								},
								Cvss:     5.4,
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
						},
					},
					{
						Name:    "kube-proxy",
						Version: "1.16.9",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:          "CVE-2020-8558",
								Cvss:         5.4,
								ScoreVersion: storage.EmbeddedVulnerability_V3,
								Severity:     storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
						},
						Vulnerabilities: []*storage.NodeVulnerability{
							{
								CveBaseInfo: &storage.CVEInfo{
									Cve: "CVE-2020-8558",
								},
								Cvss:     5.4,
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							},
						},
					},
				},
			},
		},
		{
			Name: "node2",
			Scan: &storage.NodeScan{
				Components: []*storage.EmbeddedNodeScanComponent{
					{
						Name:    "kubelet",
						Version: "1.14.3",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:          "CVE-2019-11248",
								Cvss:         6.5,
								ScoreVersion: storage.EmbeddedVulnerability_V2,
								Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							},
						},
					},
				},
			},
		},
	}
}
