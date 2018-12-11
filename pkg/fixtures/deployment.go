package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// LightweightDeployment returns a mock deployment which doesn't have all the crazy images.
func LightweightDeployment() *storage.Deployment {
	return &storage.Deployment{
		Name:        "nginx_server",
		Id:          "s79mdvmb6dsl",
		ClusterId:   "prod cluster",
		ClusterName: "prod cluster",
		Namespace:   "stackrox",
		Labels: map[string]string{
			"com.docker.stack.namespace":    "prevent",
			"com.docker.swarm.service.name": "prevent_sensor",
			"email":                         "vv@stackrox.com",
			"owner":                         "stackrox",
		},
		Containers: []*storage.Container{
			{
				Image: &storage.Image{
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
						Components: []*storage.ImageScanComponent{
							{
								Name: "name",
								Vulns: []*storage.Vulnerability{
									{
										Cve:     "cve",
										Cvss:    5,
										Summary: "Vuln summary",
									},
								},
							},
						},
					},
				},
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

// GetDeployment returns a Mock Deployment
func GetDeployment() *storage.Deployment {
	dep := LightweightDeployment()
	dep.Containers = append(dep.Containers, &storage.Container{Image: GetImage()})
	return dep
}
