package scorer

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoconv"
)

// GetMockImages returns a slice of mock images
func GetMockImages() []*storage.Image {
	return []*storage.Image{
		GetMockImage(),
	}
}

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
							Cve:  "CVE-2019-0001",
							Cvss: 5,
						},
						{
							Cve:  "CVE-2019-0002",
							Cvss: 5,
						},
						{
							Cve:  "RHSA-2019:0002",
							Cvss: 10,
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
