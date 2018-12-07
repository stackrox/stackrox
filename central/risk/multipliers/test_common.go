package multipliers

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
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
								Name:    "comp1",
								Version: "1.1.1",
								Vulns: []*storage.Vulnerability{
									{
										Cvss: 5,
									},
									{
										Cvss: 5,
									},
								},
							},
							{
								Name:    "comp1",
								Version: "1.1.1",
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
				Image: &storage.Image{
					Metadata: &storage.ImageMetadata{
						V1: &storage.V1Metadata{
							Created: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(90 * 24 * time.Hour))),
						},
					},
				},
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
