package fixtures

import "github.com/stackrox/rox/generated/api/v1"

// GetPolicy returns a Mock Policy
func GetPolicy() *v1.Policy {
	return &v1.Policy{
		Id:          "b3523d84-ac1a-4daa-a908-62d196c5a741",
		Name:        "Vulnerable Container",
		Categories:  []string{"Image Assurance", "Privileges Capabilities", "Container Configuration"},
		Description: "Alert if the container contains vulnerabilities",
		Severity:    v1.Severity_LOW_SEVERITY,
		Rationale:   "This is the rationale",
		Remediation: "This is the remediation",
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
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Registry:  "docker.io",
				Namespace: "stackrox",
				Repo:      "nginx",
				Tag:       "1.10",
			},
			SetImageAgeDays: &v1.PolicyFields_ImageAgeDays{
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
			Component: &v1.Component{
				Name:    "berkeley*",
				Version: ".*",
			},
			SetScanAgeDays: &v1.PolicyFields_ScanAgeDays{
				ScanAgeDays: 10,
			},
			Env: &v1.KeyValuePolicy{
				Key:   "key",
				Value: "value",
			},
			Command:   "cmd ",
			Args:      "arg1 arg2 arg3",
			Directory: "/directory",
			User:      "root",
			VolumePolicy: &v1.VolumePolicy{
				Name:        "name",
				Source:      "10.0.0.1/export",
				Destination: "/etc/network",
				SetReadOnly: &v1.VolumePolicy_ReadOnly{
					ReadOnly: true,
				},
				Type: "nfs",
			},
			PortPolicy: &v1.PortPolicy{
				Port:     8080,
				Protocol: "tcp",
			},
			AddCapabilities:  []string{"ADD1", "ADD2"},
			DropCapabilities: []string{"DROP1", "DROP2"},
			SetPrivileged: &v1.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
}
