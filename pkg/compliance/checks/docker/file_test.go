package docker

import (
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditCheck(t *testing.T) {
	cases := []struct {
		name   string
		file   *compliance.File
		status storage.ComplianceState
	}{
		{
			name: "CIS_Docker_v1_2_0:1_2_3",
			file: &compliance.File{
				Path:    auditFile,
				Content: []byte("/usr/bin/docker.service\n/usr/bin/dockerd"),
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:1_2_3",
			file: &compliance.File{
				Path:    auditFile,
				Content: []byte("/etc/default/docker"),
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(strings.Replace(c.name, ":", "-", -1), func(t *testing.T) {
			t.Parallel()
			standard := standards.NodeChecks[standards.CISDocker]
			require.NotNil(t, standard)
			check := standard[c.name]
			require.NotNil(t, check)

			allFiles := map[string]*compliance.File{
				"/usr/bin/docker.service": {Path: "/usr/bin/docker.service"},
				"/usr/bin/dockerd":        {Path: "/usr/bin/dockerd"},
				"/etc/default/docker":     {Path: "/etc/default/docker"},
				c.file.Path:               c.file,
			}

			mockNodeData := &standards.ComplianceData{
				Files: allFiles,
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
		})
	}
}
