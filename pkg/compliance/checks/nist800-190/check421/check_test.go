package check421

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	localIPNet = registry.NetIPNet(*netutil.MustParseCIDR("127.0.0.0/16"))

	nonLocalIPNet = registry.NetIPNet(*netutil.MustParseCIDR("0.0.0.0/24"))
)

func TestDockerInfoBasedChecks(t *testing.T) {
	cases := []struct {
		name   string
		info   types.Info
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
			info: types.Info{
				RegistryConfig: &registry.ServiceConfig{
					InsecureRegistryCIDRs: []*registry.NetIPNet{&localIPNet},
				},
			},
			cri: &compliance.ContainerRuntimeInfo{
				InsecureRegistries: &compliance.InsecureRegistriesConfig{
					InsecureCidrs: []string{localIPNet.String()},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: standards.NIST800190CheckName("4_2_1"),
			info: types.Info{
				RegistryConfig: &registry.ServiceConfig{
					InsecureRegistryCIDRs: []*registry.NetIPNet{&localIPNet, &nonLocalIPNet},
				},
			},
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
