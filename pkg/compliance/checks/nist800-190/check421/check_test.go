package check421

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	localIPNet = "127.0.0.0/16"
)

func TestDockerInfoBasedChecks(t *testing.T) {
	cases := []struct {
		name   string
		cri    *compliance.ContainerRuntimeInfo
		status storage.ComplianceState
	}{
		{
			name:   standards.NIST800190CheckName("4_2_1"),
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			cri:    &compliance.ContainerRuntimeInfo{},
		},
		{
			name: standards.NIST800190CheckName("4_2_1"),
			cri: &compliance.ContainerRuntimeInfo{
				InsecureRegistries: &compliance.InsecureRegistriesConfig{
					InsecureCidrs: []string{localIPNet},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name:   standards.NIST800190CheckName("4_2_1"),
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(strings.Replace(c.name, ":", "-", -1), func(t *testing.T) {
			t.Parallel()

			checks := standards.NodeChecks[standards.NIST800190]
			require.NotNil(t, checks)
			check := checks[c.name]
			require.NotNil(t, check)

			mockNodeData := &standards.ComplianceData{
				ContainerRuntimeInfo: c.cri,
			}

			checkResults := check.CheckFunc(mockNodeData)

			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
		})
	}
}
