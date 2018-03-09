package fixtures

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
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
			Name:        "Vulnerable Container",
			Categories:  []string{"Image Assurance", "Privileges Capabilities", "Container Configuration"},
			Description: "Alert if the container contains vulnerabilities",
			Severity:    v1.Severity_LOW_SEVERITY,
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
				Env: &v1.ConfigurationPolicy_EnvironmentPolicy{
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
				Selinux: &v1.PrivilegePolicy_SELinuxPolicy{
					User:  "user",
					Role:  "role",
					Type:  "type",
					Level: "level",
				},
			},
		},
		Deployment: &v1.Deployment{
			Name:      "nginx_server",
			Id:        "s79mdvmb6dsl",
			ClusterId: "prod cluster",
			Containers: []*v1.Container{
				{
					Image: &v1.Image{
						Name: &v1.ImageName{
							Sha:      "SHA",
							Registry: "docker.io",
							Remote:   "library/nginx",
							Tag:      "latest",
						},
					},
				},
			},
		},
	}
}
