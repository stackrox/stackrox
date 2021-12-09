package docker

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerImagesChecks(t *testing.T) {
	cases := []struct {
		name   string
		image  types.ImageWrap
		status storage.ComplianceState
	}{
		{
			name: "CIS_Docker_v1_2_0:4_6",
			image: types.ImageWrap{
				Image: types.ImageInspect{
					Config: &types.Config{},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_6",
			image: types.ImageWrap{
				Image: types.ImageInspect{
					Config: &types.Config{
						Healthcheck: &container.HealthConfig{},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:4_9",
			image: types.ImageWrap{
				History: []image.HistoryResponseItem{
					{
						CreatedBy: "/bin/sh -c #(nop) WORKDIR /usr/share/grafana",
					},
					{
						CreatedBy: "add file: hello",
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_9",
			image: types.ImageWrap{
				History: []image.HistoryResponseItem{
					{
						CreatedBy: "/bin/sh -c #(nop) WORKDIR /usr/share/grafana",
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:4_7",
			image: types.ImageWrap{
				History: []image.HistoryResponseItem{
					{
						CreatedBy: "/bin/sh -c #(nop) WORKDIR /usr/share/grafana",
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:4_7",
			image: types.ImageWrap{
				History: []image.HistoryResponseItem{
					{
						CreatedBy: "/bin/sh -c #(nop) apk update",
					},
				},
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
				DockerData: &types.Data{
					Images: []types.ImageWrap{
						c.image,
					},
				},
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
		})
	}
}
