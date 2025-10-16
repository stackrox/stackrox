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
		storage.Image_builder{
			Name: storage.ImageName_builder{
				FullName: "docker.io/library/nginx:1.10",
				Registry: "docker.io",
				Remote:   "library/nginx",
				Tag:      "1.10",
			}.Build(),
			Scan: storage.ImageScan_builder{
				Components: []*storage.EmbeddedImageScanComponent{
					storage.EmbeddedImageScanComponent_builder{
						Name:    "comp1",
						Version: "1.1.1",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2019-0001",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
							}.Build(),
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2019-0002",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "comp1",
						Version: "1.1.1",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:          "CVE-2019-0001",
								Cvss:         5,
								ScoreVersion: storage.EmbeddedVulnerability_V3,
								Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							}.Build(),
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2019-0002",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
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
		}.Build(),
		storage.Image_builder{
			Metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(90 * 24 * time.Hour))),
				}.Build(),
			}.Build(),
		}.Build(),
	}
}

// GetMockImagesV2 returns a slice of mock images v2
func GetMockImagesV2() []*storage.ImageV2 {
	return []*storage.ImageV2{
		storage.ImageV2_builder{
			Name: storage.ImageName_builder{
				FullName: "docker.io/library/nginx:1.10",
				Registry: "docker.io",
				Remote:   "library/nginx",
				Tag:      "1.10",
			}.Build(),
			Scan: storage.ImageScan_builder{
				Components: []*storage.EmbeddedImageScanComponent{
					storage.EmbeddedImageScanComponent_builder{
						Name:    "comp1",
						Version: "1.1.1",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2019-0001",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
							}.Build(),
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2019-0002",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
					storage.EmbeddedImageScanComponent_builder{
						Name:    "comp1",
						Version: "1.1.1",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:          "CVE-2019-0001",
								Cvss:         5,
								ScoreVersion: storage.EmbeddedVulnerability_V3,
								Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							}.Build(),
							storage.EmbeddedVulnerability_builder{
								Cve:      "CVE-2019-0002",
								Cvss:     5,
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
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
		}.Build(),
		storage.ImageV2_builder{
			Metadata: storage.ImageMetadata_builder{
				V1: storage.V1Metadata_builder{
					Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(90 * 24 * time.Hour))),
				}.Build(),
			}.Build(),
		}.Build(),
	}
}

// GetMockDeployment returns a mock deployment
func GetMockDeployment() *storage.Deployment {
	return storage.Deployment_builder{
		Name:                         "mock-deployment",
		ServiceAccount:               "service-account",
		ClusterId:                    "cluster",
		Namespace:                    "namespace",
		AutomountServiceAccountToken: true,
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
				Name: "containerName",
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
				Image: types.ToContainerImage(GetMockImages()[0]),
			}.Build(),
			storage.Container_builder{
				Name:  "Container2",
				Image: types.ToContainerImage(GetMockImages()[1]),
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

// GetMockNodes returns a slice of mock nodes
func GetMockNodes() []*storage.Node {
	return []*storage.Node{
		storage.Node_builder{
			Name: "node1",
			Scan: storage.NodeScan_builder{
				Components: []*storage.EmbeddedNodeScanComponent{
					storage.EmbeddedNodeScanComponent_builder{
						Name:    "kubelet",
						Version: "1.16.9",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:          "CVE-2020-8558",
								Cvss:         5.4,
								ScoreVersion: storage.EmbeddedVulnerability_V3,
								Severity:     storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							}.Build(),
						},
						Vulnerabilities: []*storage.NodeVulnerability{
							storage.NodeVulnerability_builder{
								CveBaseInfo: storage.CVEInfo_builder{
									Cve: "CVE-2020-8558",
								}.Build(),
								Cvss:     5.4,
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
					storage.EmbeddedNodeScanComponent_builder{
						Name:    "kube-proxy",
						Version: "1.16.9",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:          "CVE-2020-8558",
								Cvss:         5.4,
								ScoreVersion: storage.EmbeddedVulnerability_V3,
								Severity:     storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							}.Build(),
						},
						Vulnerabilities: []*storage.NodeVulnerability{
							storage.NodeVulnerability_builder{
								CveBaseInfo: storage.CVEInfo_builder{
									Cve: "CVE-2020-8558",
								}.Build(),
								Cvss:     5.4,
								Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		}.Build(),
		storage.Node_builder{
			Name: "node2",
			Scan: storage.NodeScan_builder{
				Components: []*storage.EmbeddedNodeScanComponent{
					storage.EmbeddedNodeScanComponent_builder{
						Name:    "kubelet",
						Version: "1.14.3",
						Vulns: []*storage.EmbeddedVulnerability{
							storage.EmbeddedVulnerability_builder{
								Cve:          "CVE-2019-11248",
								Cvss:         6.5,
								ScoreVersion: storage.EmbeddedVulnerability_V2,
								Severity:     storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		}.Build(),
	}
}
