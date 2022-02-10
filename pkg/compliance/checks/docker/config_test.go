package docker

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwnershipAndPermissionChecks(t *testing.T) {
	cases := []struct {
		name   string
		file   *compliance.File
		status storage.ComplianceState
	}{
		{
			name: "CIS_Docker_v1_2_0:3_1",
			file: &compliance.File{
				Path:      "docker.service",
				GroupName: "root",
				UserName:  "root",
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:3_1",
			file: &compliance.File{
				Path:      "docker.service",
				GroupName: "docker",
				UserName:  "root",
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:3_1",
			file: &compliance.File{
				Path:      "docker.service",
				GroupName: "root",
				UserName:  "docker",
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:3_2",
			file: &compliance.File{
				Path:        "docker.service",
				Permissions: 0644,
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:3_2",
			file: &compliance.File{
				Path:        "docker.service",
				Permissions: 0643,
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:3_2",
			file: &compliance.File{
				Path:        "docker.service",
				Permissions: 0645,
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

			mockNodeData := &standards.ComplianceData{
				SystemdFiles: map[string]*compliance.File{
					c.file.Path: c.file,
				},
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
		})
	}
}
