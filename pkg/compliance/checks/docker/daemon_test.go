package docker

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	internalTypes "github.com/stackrox/stackrox/pkg/docker/types"
	"github.com/stackrox/stackrox/pkg/netutil"
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
			name: "CIS_Docker_v1_2_0:2_5",
			info: types.Info{
				Driver: "aufs",
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:2_5",
			info: types.Info{
				Driver: "overlay2",
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:2_15",
			info: types.Info{
				SecurityOptions: []string{"hello", "seccomp=default"},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_NOTE,
		},
		{
			name: "CIS_Docker_v1_2_0:2_16",
			info: types.Info{
				ExperimentalBuild: true,
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name:   "CIS_Docker_v1_2_0:2_16",
			info:   types.Info{},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name:   "CIS_Docker_v1_2_0:2_4",
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
			cri:    &compliance.ContainerRuntimeInfo{},
		},
		{
			name: "CIS_Docker_v1_2_0:2_4",
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
			name: "CIS_Docker_v1_2_0:2_4",
			info: types.Info{
				RegistryConfig: &registry.ServiceConfig{
					InsecureRegistryCIDRs: []*registry.NetIPNet{&localIPNet, &nonLocalIPNet},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:2_13",
			info: types.Info{
				LiveRestoreEnabled: true,
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name:   "CIS_Docker_v1_2_0:2_13",
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:2_12",
			info: types.Info{
				LoggingDriver: "json-file",
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name:   "CIS_Docker_v1_2_0:2_12",
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(strings.Replace(c.name, ":", "-", -1), func(t *testing.T) {
			t.Parallel()

			checks := standards.NodeChecks[standards.CISDocker]
			require.NotNil(t, checks)
			check := checks[c.name]
			require.NotNil(t, check)

			mockNodeData := &standards.ComplianceData{
				DockerData: &internalTypes.Data{
					Info: c.info,
				},
				ContainerRuntimeInfo: c.cri,
			}

			checkResults := check.CheckFunc(mockNodeData)

			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
		})
	}
}
