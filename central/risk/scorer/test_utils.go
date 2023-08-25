package scorer

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoconv"
)

// GetMockImage returns a mock image
func GetMockImage() *storage.Image {
	return &storage.Image{
		Id: "ImageID",
		Name: &storage.ImageName{
			FullName: "docker.io/library/nginx:1.10",
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "ComponentX",
					Version: "v1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:          "CVE-2019-0001",
							Cvss:         5,
							ScoreVersion: storage.EmbeddedVulnerability_V3,
							Severity:     storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "CVE-2019-0002",
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:          "RHSA-2019:0002",
							Cvss:         10,
							ScoreVersion: storage.EmbeddedVulnerability_V3,
							Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
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
	}
}

// GetMockDeployment returns a mock deployment
func GetMockDeployment() *storage.Deployment {
	return &storage.Deployment{
		Id:        "DeploymentID",
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
				Image: types.ToContainerImage(GetMockImage()),
			},
			{
				Name: "second",
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

// GetMockNode returns a mock node
func GetMockNode() *storage.Node {
	return &storage.Node{
		Id:   "nodeID",
		Name: "node1",
		Scan: &storage.NodeScan{
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "ComponentX",
					Version: "v1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "CVE-2019-0001",
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "CVE-2019-0002",
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
					},
					Vulnerabilities: []*storage.NodeVulnerability{
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "CVE-2019-0001",
							},
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "CVE-2019-0002",
							},
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
		},
	}
}
