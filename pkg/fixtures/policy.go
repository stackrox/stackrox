package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetPolicy returns a Mock Policy
func GetPolicy() *storage.Policy {
	return &storage.Policy{
		Id:          "b3523d84-ac1a-4daa-a908-62d196c5a741",
		Name:        "Vulnerable Container",
		Categories:  []string{"Image Assurance", "Privileges Capabilities", "Container Configuration"},
		Description: "Alert if the container contains vulnerabilities",
		Severity:    storage.Severity_LOW_SEVERITY,
		Rationale:   "This is the rationale",
		Remediation: "This is the remediation",
		Scope: []*storage.Scope{
			{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
				Label: &storage.Scope_Label{
					Key:   "com.docker.stack.namespace",
					Value: "prevent",
				},
			},
		},
		Fields: &storage.PolicyFields{
			ImageName: &storage.ImageNamePolicy{
				Registry: "docker.io",
				Remote:   "stackrox/nginx",
				Tag:      "1.10",
			},
			SetImageAgeDays: &storage.PolicyFields_ImageAgeDays{
				ImageAgeDays: 30,
			},
			LineRule: &storage.DockerfileLineRuleField{
				Instruction: "VOLUME",
				Value:       "/etc/*",
			},
			Cvss: &storage.NumericalPolicy{
				Op:    storage.Comparator_GREATER_THAN_OR_EQUALS,
				Value: 5,
			},
			Cve: "CVE-1234",
			Component: &storage.Component{
				Name:    "berkeley*",
				Version: ".*",
			},
			SetScanAgeDays: &storage.PolicyFields_ScanAgeDays{
				ScanAgeDays: 10,
			},
			Env: &storage.KeyValuePolicy{
				Key:   "key",
				Value: "value",
			},
			Command:   "cmd ",
			Args:      "arg1 arg2 arg3",
			Directory: "/directory",
			User:      "root",
			VolumePolicy: &storage.VolumePolicy{
				Name:        "name",
				Source:      "10.0.0.1/export",
				Destination: "/etc/network",
				SetReadOnly: &storage.VolumePolicy_ReadOnly{
					ReadOnly: true,
				},
				Type: "nfs",
			},
			PortPolicy: &storage.PortPolicy{
				Port:     8080,
				Protocol: "tcp",
			},
			AddCapabilities:  []string{"ADD1", "ADD2"},
			DropCapabilities: []string{"DROP1", "DROP2"},
			SetPrivileged: &storage.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
}
