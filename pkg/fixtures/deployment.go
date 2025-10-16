package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	types2 "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

// LightweightDeploymentImage returns the full images referenced by GetLightweightDeployment.
func LightweightDeploymentImage() *storage.Image {
	return storage.Image_builder{
		Id: "sha256:SHA1",
		Name: storage.ImageName_builder{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "1.10",
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "ADD",
						Value:       "FILE:blah",
					}.Build(),
				},
			}.Build(),
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScanTime: protocompat.TimestampNow(),
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{
					Name: "name",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:     "cve",
							Cvss:    5,
							Summary: "Vuln summary",
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}.Build()
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
	return storage.Deployment_builder{
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
			storage.Container_builder{
				Name:  "nginx110container",
				Image: types2.ToContainerImage(LightweightDeploymentImage()),
				SecurityContext: storage.SecurityContext_builder{
					Privileged:       true,
					AddCapabilities:  []string{"SYS_ADMIN"},
					DropCapabilities: []string{"SYS_MODULE"},
				}.Build(),
				Resources: storage.Resources_builder{CpuCoresRequest: 0.9}.Build(),
				Config: storage.ContainerConfig_builder{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						storage.ContainerConfig_EnvironmentConfig_builder{
							Key:   "envkey",
							Value: "envvalue",
						}.Build(),
					},
				}.Build(),
				Volumes: []*storage.Volume{
					storage.Volume_builder{
						Name:        "vol1",
						Source:      "/vol1",
						Destination: "/vol2",
						Type:        "host",
						ReadOnly:    true,
					}.Build(),
				},
				Secrets: []*storage.EmbeddedSecret{
					storage.EmbeddedSecret_builder{
						Name: "secretname",
						Path: "/var/lib/stackrox",
					}.Build(),
				},
			}.Build(),
		},
		Priority: 1,
	}.Build()
}

// DuplicateImageDeployment returns a mock deployment with two containers that have the same image.
func DuplicateImageDeployment() *storage.Deployment {
	container := &storage.Container{}
	container.SetName("nginx-1")
	container.SetImage(types2.ToContainerImage(LightweightDeploymentImage()))
	container2 := &storage.Container{}
	container2.SetName("nginx-2")
	container2.SetImage(types2.ToContainerImage(LightweightDeploymentImage()))
	container3 := &storage.Container{}
	container3.SetName("supervulnerable")
	container3.SetImage(types2.ToContainerImage(GetImage()))
	deployment := &storage.Deployment{}
	deployment.SetName("nginx_server")
	deployment.SetId(fixtureconsts.Deployment1)
	deployment.SetClusterId(fixtureconsts.Cluster1)
	deployment.SetClusterName("prod cluster")
	deployment.SetNamespace("stackrox")
	deployment.SetContainers([]*storage.Container{
		container,
		container2,
		container3,
	})
	deployment.SetPriority(1)
	return deployment
}

// GetDeployment returns a Mock Deployment.
func GetDeployment() *storage.Deployment {
	dep := LightweightDeployment()
	container := &storage.Container{}
	container.SetName("supervulnerable")
	container.SetImage(types2.ToContainerImage(GetImage()))
	dep.SetContainers(append(dep.GetContainers(), container))
	return dep
}

// GetScopedDeployment returns a Mock Deployment with the provided ID and scoping info for testing purposes.
func GetScopedDeployment(ID string, clusterID string, namespace string) *storage.Deployment {
	deployment := LightweightDeployment()
	deployment.SetId(ID)
	deployment.SetClusterId(clusterID)
	deployment.SetNamespace(namespace)
	return deployment
}

// GetDeploymentWithImage returns a Mock Deployment with specified image.
func GetDeploymentWithImage(cluster, namespace string, image *storage.Image) *storage.Deployment {
	dep := LightweightDeployment()
	dep.SetId(uuid.NewV4().String())
	dep.SetClusterName(cluster)
	dep.SetClusterId(cluster)
	dep.SetNamespace(namespace)
	dep.SetNamespaceId(cluster + namespace)
	container := &storage.Container{}
	container.SetName("supervulnerable")
	container.SetImage(types2.ToContainerImage(image))
	dep.SetContainers(append(dep.GetContainers(), container))
	return dep
}

// GetDeploymentWithImageV2 returns a Mock Deployment with specified ImageV2.
func GetDeploymentWithImageV2(cluster, namespace string, image *storage.ImageV2) *storage.Deployment {
	dep := LightweightDeployment()
	dep.SetId(uuid.NewV4().String())
	dep.SetClusterName(cluster)
	dep.SetClusterId(cluster)
	dep.SetNamespace(namespace)
	dep.SetNamespaceId(cluster + namespace)
	container := &storage.Container{}
	container.SetName("supervulnerable")
	container.SetImage(types2.ToContainerImageV2(image))
	dep.SetContainers(append(dep.GetContainers(), container))
	return dep
}
