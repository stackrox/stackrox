package scorer

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoconv"
)

// GetMockImage returns a mock image
func GetMockImage() *storage.Image {
	return storage.Image_builder{
		Id: "ImageID",
		Name: storage.ImageName_builder{
			FullName: "docker.io/library/nginx:1.10",
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		}.Build(),
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{
					Name:    "ComponentX",
					Version: "v1",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:          "CVE-2019-0001",
							Cvss:         5,
							ScoreVersion: storage.EmbeddedVulnerability_V3,
							Severity:     storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:      "CVE-2019-0002",
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:          "RHSA-2019:0002",
							Cvss:         10,
							ScoreVersion: storage.EmbeddedVulnerability_V3,
							Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(180 * 24 * time.Hour))),
			}.Build(),
		}.Build(),
	}.Build()
}

// GetMockImageV2 returns a mock image v2
func GetMockImageV2() *storage.ImageV2 {
	return storage.ImageV2_builder{
		Id: "ImageID",
		Name: storage.ImageName_builder{
			FullName: "docker.io/library/nginx:1.10",
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		}.Build(),
		Scan: storage.ImageScan_builder{
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{
					Name:    "ComponentX",
					Version: "v1",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:          "CVE-2019-0001",
							Cvss:         5,
							ScoreVersion: storage.EmbeddedVulnerability_V3,
							Severity:     storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:      "CVE-2019-0002",
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:          "RHSA-2019:0002",
							Cvss:         10,
							ScoreVersion: storage.EmbeddedVulnerability_V3,
							Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(180 * 24 * time.Hour))),
			}.Build(),
		}.Build(),
	}.Build()
}

// GetMockDeployment returns a mock deployment
func GetMockDeployment() *storage.Deployment {
	return storage.Deployment_builder{
		Id:        "DeploymentID",
		ClusterId: "cluster",
		Ports: []*storage.PortConfig{
			storage.PortConfig_builder{
				Name:          "Port1",
				ContainerPort: 22,
				Exposure:      storage.PortConfig_EXTERNAL,
			}.Build(),
			storage.PortConfig_builder{
				Name:          "Port2",
				ContainerPort: 23,
				Exposure:      storage.PortConfig_INTERNAL,
			}.Build(),
			storage.PortConfig_builder{
				Name:          "Port3",
				ContainerPort: 24,
				Exposure:      storage.PortConfig_NODE,
			}.Build(),
		},
		Containers: []*storage.Container{
			storage.Container_builder{
				Name: "nginx",
				Volumes: []*storage.Volume{
					storage.Volume_builder{
						Name:     "readonly",
						ReadOnly: true,
					}.Build(),
				},
				Secrets: []*storage.EmbeddedSecret{
					storage.EmbeddedSecret_builder{
						Name: "secret",
					}.Build(),
				},
				SecurityContext: storage.SecurityContext_builder{
					AddCapabilities: []string{
						"ALL",
					},
					Privileged: true,
				}.Build(),
				Image: types.ToContainerImage(GetMockImage()),
			}.Build(),
			storage.Container_builder{
				Name: "second",
				Volumes: []*storage.Volume{
					storage.Volume_builder{
						Name: "rw volume",
					}.Build(),
				},
				SecurityContext: &storage.SecurityContext{},
			}.Build(),
		},
	}.Build()
}

// GetMockNode returns a mock node
func GetMockNode() *storage.Node {
	return storage.Node_builder{
		Id:   "nodeID",
		Name: "node1",
		Scan: storage.NodeScan_builder{
			Components: []*storage.EmbeddedNodeScanComponent{
				storage.EmbeddedNodeScanComponent_builder{
					Name:    "ComponentX",
					Version: "v1",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:      "CVE-2019-0001",
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:      "CVE-2019-0002",
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						}.Build(),
					},
					Vulnerabilities: []*storage.NodeVulnerability{
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "CVE-2019-0001",
							}.Build(),
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						}.Build(),
						storage.NodeVulnerability_builder{
							CveBaseInfo: storage.CVEInfo_builder{
								Cve: "CVE-2019-0002",
							}.Build(),
							Cvss:     5,
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}.Build()
}
