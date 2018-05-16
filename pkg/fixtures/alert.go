package fixtures

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	ptypes "github.com/gogo/protobuf/types"
)

// GetAlert returns a Mock Alert
func GetAlert() *v1.Alert {
	return &v1.Alert{
		Id: "Alert1",
		Violations: []*v1.Alert_Violation{
			{
				Message: "Deployment is affected by 'CVE-2017-15804'",
			},
			{
				Message: "Deployment is affected by 'CVE-2017-15670'",
			},
		},
		Time: ptypes.TimestampNow(),
		Policy: &v1.Policy{
			Id:          "b3523d84-ac1a-4daa-a908-62d196c5a741",
			Name:        "Vulnerable Container",
			Categories:  []string{"Image Assurance", "Privileges Capabilities", "Container Configuration"},
			Description: "Alert if the container contains vulnerabilities",
			Severity:    v1.Severity_LOW_SEVERITY,
			Scope: []*v1.Scope{
				{
					Cluster:   "prod cluster",
					Namespace: "stackrox",
					Label: &v1.Scope_Label{
						Key:   "com.docker.stack.namespace",
						Value: "prevent",
					},
				},
			},
			ImagePolicy: &v1.ImagePolicy{
				ImageName: &v1.ImageNamePolicy{
					Registry:  "docker.io",
					Namespace: "stackrox",
					Repo:      "nginx",
					Tag:       "1.10",
				},
				SetImageAgeDays: &v1.ImagePolicy_ImageAgeDays{
					ImageAgeDays: 30,
				},
				LineRule: &v1.DockerfileLineRuleField{
					Instruction: "VOLUME",
					Value:       "/etc/*",
				},
				Cvss: &v1.NumericalPolicy{
					Op:     v1.Comparator_GREATER_THAN_OR_EQUALS,
					MathOp: v1.MathOP_MAX,
					Value:  5,
				},
				Cve: "CVE-1234",
				Component: &v1.ImagePolicy_Component{
					Name:    "berkeley*",
					Version: ".*",
				},
				SetScanAgeDays: &v1.ImagePolicy_ScanAgeDays{
					ScanAgeDays: 10,
				},
			},
			ConfigurationPolicy: &v1.ConfigurationPolicy{
				Env: &v1.ConfigurationPolicy_KeyValuePolicy{
					Key:   "key",
					Value: "value",
				},
				Command:   "cmd ",
				Args:      "arg1 arg2 arg3",
				Directory: "/directory",
				User:      "root",
				VolumePolicy: &v1.ConfigurationPolicy_VolumePolicy{
					Name:        "name",
					Source:      "10.0.0.1/export",
					Destination: "/etc/network",
					SetReadOnly: &v1.ConfigurationPolicy_VolumePolicy_ReadOnly{
						ReadOnly: true,
					},
					Type: "nfs",
				},
				PortPolicy: &v1.ConfigurationPolicy_PortPolicy{
					Port:     8080,
					Protocol: "tcp",
				},
			},
			PrivilegePolicy: &v1.PrivilegePolicy{
				AddCapabilities:  []string{"ADD1", "ADD2"},
				DropCapabilities: []string{"DROP1", "DROP2"},
				SetPrivileged: &v1.PrivilegePolicy_Privileged{
					Privileged: true,
				},
			},
		},
		Deployment: &v1.Deployment{
			Name:        "nginx_server",
			Id:          "s79mdvmb6dsl",
			ClusterId:   "prod cluster",
			ClusterName: "prod cluster",
			Namespace:   "stackrox",
			Labels: []*v1.Deployment_KeyValue{
				{
					Key:   "com.docker.stack.namespace",
					Value: "prevent",
				},
				{
					Key:   "com.docker.swarm.service.name",
					Value: "prevent_sensor",
				},
			},
			Containers: []*v1.Container{
				{
					Image: &v1.Image{
						Name: &v1.ImageName{
							Sha:      "sha256:SHA1",
							Registry: "docker.io",
							Remote:   "library/nginx",
							Tag:      "1.10",
						},
						Metadata: &v1.ImageMetadata{
							Layers: []*v1.ImageLayer{
								{
									Instruction: "ADD",
									Value:       "FILE:blah",
								},
							},
						},
						Scan: &v1.ImageScan{
							Components: []*v1.ImageScanComponent{
								{
									Name: "name",
									Vulns: []*v1.Vulnerability{
										{
											Cve:     "cve",
											Cvss:    10,
											Summary: "Vuln summary",
										},
									},
								},
							},
						},
					},
					SecurityContext: &v1.SecurityContext{
						Privileged:       true,
						AddCapabilities:  []string{"SYS_ADMIN"},
						DropCapabilities: []string{"SYS_MODULE"},
					},
					Config: &v1.ContainerConfig{
						Env: []*v1.ContainerConfig_EnvironmentConfig{
							{
								Key:   "envkey",
								Value: "envvalue",
							},
						},
					},
					Volumes: []*v1.Volume{
						{
							Name:        "vol1",
							Source:      "/vol1",
							Destination: "/vol2",
							Type:        "host",
							ReadOnly:    true,
						},
					},
				},
				{
					Image: &v1.Image{
						Name: &v1.ImageName{
							Sha:      "sha256:SHA2",
							Registry: "stackrox.io",
							Remote:   "srox/mongo",
							Tag:      "latest",
						},
					},
				},
			},
		},
	}
}
