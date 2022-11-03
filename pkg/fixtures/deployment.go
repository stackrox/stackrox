package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	types2 "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/uuid"
)

// LightweightDeploymentImage returns the full images referenced by GetLightweightDeployment.
func LightweightDeploymentImage() *storage.Image {
	return &storage.Image{
		Id: "sha256:SHA1",
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: []*storage.ImageLayer{
					{
						Instruction: "ADD",
						Value:       "FILE:blah",
					},
				},
			},
		},
		Scan: &storage.ImageScan{
			ScanTime: types.TimestampNow(),
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name: "name",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:     "cve",
							Cvss:    5,
							Summary: "Vuln summary",
						},
					},
				},
			},
		},
	}
}

// DeploymentImages returns the full images referenced by GetDeployment.
func DeploymentImages() []*storage.Image {
	return []*storage.Image{
		LightweightDeploymentImage(),
		GetImage(),
	}
}

// LightweightDeployment returns a mock deployment which doesn't have all the crazy images.
func LightweightDeployment() *storage.Deployment {
	return &storage.Deployment{
		Name:        "nginx_server",
		Id:          fixtureconsts.Deployment1,
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "prod cluster",
		Namespace:   "stackrox",
		Annotations: map[string]string{
			"team": "stackrox",
		},
		Labels: map[string]string{
			"com.docker.stack.namespace":    "prevent",
			"com.docker.swarm.service.name": "prevent_sensor",
			"email":                         "vv@stackrox.com",
			"owner":                         "stackrox",
		},
		PodLabels: map[string]string{
			"app": "nginx",
		},
		Containers: []*storage.Container{
			{
				Name:  "nginx110container",
				Image: types2.ToContainerImage(LightweightDeploymentImage()),
				SecurityContext: &storage.SecurityContext{
					Privileged:       true,
					AddCapabilities:  []string{"SYS_ADMIN"},
					DropCapabilities: []string{"SYS_MODULE"},
				},
				Resources: &storage.Resources{CpuCoresRequest: 0.9},
				Config: &storage.ContainerConfig{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{
							Key:   "envkey",
							Value: "envvalue",
						},
					},
				},
				Volumes: []*storage.Volume{
					{
						Name:        "vol1",
						Source:      "/vol1",
						Destination: "/vol2",
						Type:        "host",
						ReadOnly:    true,
					},
				},
				Secrets: []*storage.EmbeddedSecret{
					{
						Name: "secretname",
						Path: "/var/lib/stackrox",
					},
				},
			},
		},
	}
}

// GetDeployment returns a Mock Deployment.
func GetDeployment() *storage.Deployment {
	dep := LightweightDeployment()
	dep.Containers = append(dep.Containers, &storage.Container{Name: "supervulnerable", Image: types2.ToContainerImage(GetImage())})
	return dep
}

// GetScopedDeployment returns a Mock Deployment with the provided ID and scoping info for testing purposes.
func GetScopedDeployment(ID string, clusterID string, namespace string) *storage.Deployment {
	deployment := LightweightDeployment()
	deployment.Id = ID
	deployment.ClusterId = clusterID
	deployment.Namespace = namespace
	return deployment
}

// GetDeploymentWithImage returns a Mock Deployment with specified image.
func GetDeploymentWithImage(cluster, namespace string, image *storage.Image) *storage.Deployment {
	dep := LightweightDeployment()
	dep.Id = uuid.NewV4().String()
	dep.ClusterName = cluster
	dep.ClusterId = cluster
	dep.Namespace = namespace
	dep.NamespaceId = cluster + namespace
	dep.Containers = append(dep.Containers, &storage.Container{Name: "supervulnerable", Image: types2.ToContainerImage(image)})
	return dep
}
