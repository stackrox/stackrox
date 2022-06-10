package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	types2 "github.com/stackrox/rox/pkg/images/types"
)

// LightweightDeploymentImage returns the full images referenced by GetLightweightDeployment
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

// DeploymentImages returns the full images referenced by GetDeployment
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
		Id:          "s79mdvmb6dsl",
		ClusterId:   "prod cluster",
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

// GetDeployment returns a Mock Deployment
func GetDeployment() *storage.Deployment {
	dep := LightweightDeployment()
	dep.Containers = append(dep.Containers, &storage.Container{Name: "supervulnerable", Image: types2.ToContainerImage(GetImage())})
	return dep
}

func GetDeploymentCoreDNS_1_8_0(id string) *storage.Deployment {
	return &storage.Deployment{
		Id:                    id,
		Name:                  "coredns",
		Hash:                  7991865522150829945,
		Type:                  "Deployment",
		Namespace:             "kube-system",
		NamespaceId:           "317ac0e5-2ecd-4c4d-b3a1-58f9f93909a2",
		OrchestratorComponent: true,
		Replicas:              2,
		Labels:                map[string]string{"k8s-app": "kube-dns"},
		PodLabels:             map[string]string{"k8s-app": "kube-dns"},
		LabelSelector:         &storage.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}},
		Created:               &types.Timestamp{Seconds: 1643589436},
		ClusterId:             "50d16311-13cd-4690-9fd8-9c5c88f47ad2",
		ClusterName:           "remote",
		Containers: []*storage.Container{
			{
				Id: "c40f9039-15ee-47f0-a5d4-a52f28eb0318:coredns",
				Config: &storage.ContainerConfig{
					Args: []string{"-conf", "/etc/coredns/Corefile"},
				},
				Image: &storage.ContainerImage{
					Id:   GetCoreDNS_1_8_0().GetId(),
					Name: GetCoreDNS_1_8_0().GetName(),
				},
				SecurityContext: &storage.SecurityContext{
					DropCapabilities:       []string{"all"},
					AddCapabilities:        []string{"NET_BIND_SERVICE"},
					ReadOnlyRootFilesystem: true,
				},
				Volumes: []*storage.Volume{
					{
						Name:        "config-volume",
						Source:      "coredns",
						Destination: "/etc/coredns",
						ReadOnly:    true,
						Type:        "ConfigMap",
					},
				},
				Ports: []*storage.PortConfig{
					{
						Name:          "dns",
						ContainerPort: 53,
						Protocol:      "UDP",
						Exposure:      storage.PortConfig_INTERNAL,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								Level:            storage.PortConfig_INTERNAL,
								ServiceName:      "kube-dns",
								ServiceId:        "6ec876d4-a877-4b10-97a1-8ebfa718eec6",
								ServiceClusterIp: "10.10.0.10",
								ServicePort:      53,
							},
						},
					},
					{
						Name:          "dns-tcp",
						ContainerPort: 53,
						Protocol:      "TCP",
						Exposure:      storage.PortConfig_INTERNAL,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								Level:            storage.PortConfig_INTERNAL,
								ServiceName:      "kube-dns",
								ServiceId:        "6ec876d4-a877-4b10-97a1-8ebfa718eec6",
								ServiceClusterIp: "10.10.0.10",
								ServicePort:      53,
							},
						},
					},
					{
						Name:          "metrics",
						ContainerPort: 9153,
						Protocol:      "TCP",
						Exposure:      storage.PortConfig_INTERNAL,
						ExposureInfos: []*storage.PortConfig_ExposureInfo{
							{
								Level:            storage.PortConfig_INTERNAL,
								ServiceName:      "kube-dns",
								ServiceId:        "6ec876d4-a877-4b10-97a1-8ebfa718eec6",
								ServiceClusterIp: "10.10.0.10",
								ServicePort:      9153,
							},
						},
					},
				},
				Resources: &storage.Resources{
					CpuCoresRequest: 0.1,
					MemoryMbRequest: 70.0,
					MemoryMbLimit:   170.0,
				},
				Name:           "coredns",
				LivenessProbe:  &storage.LivenessProbe{Defined: true},
				ReadinessProbe: &storage.ReadinessProbe{Defined: true},
			},
		},
		Priority:                      4,
		ServiceAccount:                "",
		ServiceAccountPermissionLevel: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE,
		AutomountServiceAccountToken:  true,
		Tolerations: []*storage.Toleration{
			{
				Key:      "CriticalAddonsOnly",
				Operator: storage.Toleration_TOLERATION_OPERATOR_EXISTS,
			},
			{
				Key:         "node-role.kubernetes.io/master",
				TaintEffect: storage.TaintEffect_NO_SCHEDULE_TAINT_EFFECT,
			},
			{
				Key:         "node-role.kubernetes.io/control-plane",
				TaintEffect: storage.TaintEffect_NO_SCHEDULE_TAINT_EFFECT,
			},
		},
		Ports: []*storage.PortConfig{
			{
				Name:          "dns",
				ContainerPort: 53,
				Protocol:      "UDP",
				Exposure:      storage.PortConfig_INTERNAL,
				ExposureInfos: []*storage.PortConfig_ExposureInfo{
					{
						Level:            storage.PortConfig_INTERNAL,
						ServiceName:      "kube-dns",
						ServiceId:        "6ec876d4-a877-4b10-97a1-8ebfa718eec6",
						ServiceClusterIp: "10.10.0.10",
						ServicePort:      53,
					},
				},
			},
			{
				Name:          "dns-tcp",
				ContainerPort: 53,
				Protocol:      "TCP",
				Exposure:      storage.PortConfig_INTERNAL,
				ExposureInfos: []*storage.PortConfig_ExposureInfo{
					{
						Level:            storage.PortConfig_INTERNAL,
						ServiceName:      "kube-dns",
						ServiceId:        "6ec876d4-a877-4b10-97a1-8ebfa718eec6",
						ServiceClusterIp: "10.10.0.10",
						ServicePort:      53,
					},
				},
			},
			{
				Name:          "metrics",
				ContainerPort: 9153,
				Protocol:      "TCP",
				Exposure:      storage.PortConfig_INTERNAL,
				ExposureInfos: []*storage.PortConfig_ExposureInfo{
					{
						Level:            storage.PortConfig_INTERNAL,
						ServiceName:      "kube-dns",
						ServiceId:        "6ec876d4-a877-4b10-97a1-8ebfa718eec6",
						ServiceClusterIp: "10.10.0.10",
						ServicePort:      9153,
					},
				},
			},
		},
		StateTimestamp: 1654762976894737,
		RiskScore:      1.9846836,
	}
}

func GetScopedDeploymentNginX_xxx() *storage.Deployment {
	return nil
}
