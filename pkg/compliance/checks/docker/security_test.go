package docker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/docker/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerSecurityChecks(t *testing.T) {
	cases := []struct {
		name       string
		dockerData types.Data
		status     storage.ComplianceState
		counted    int
		outOf      int
	}{
		{
			name:    "CIS_Docker_v1_2_0:6_1",
			status:  storage.ComplianceState_COMPLIANCE_STATE_NOTE,
			counted: 2,
			outOf:   3,
			dockerData: types.Data{
				Containers: []types.ContainerJSON{
					{
						ContainerJSONBase: &types.ContainerJSONBase{
							Image: "Image one",
						},
					},
					{
						ContainerJSONBase: &types.ContainerJSONBase{
							Image: "Image two",
						},
					},
				},
				Images: []types.ImageWrap{
					{},
					{},
					{},
				},
			},
		},
		{
			name:    "CIS_Docker_v1_2_0:6_2",
			status:  storage.ComplianceState_COMPLIANCE_STATE_NOTE,
			counted: 1,
			outOf:   2,
			dockerData: types.Data{
				Containers: []types.ContainerJSON{
					{
						ContainerJSONBase: &types.ContainerJSONBase{
							State: &types.ContainerState{
								Running: true,
							},
						},
					},
					{
						ContainerJSONBase: &types.ContainerJSONBase{
							State: &types.ContainerState{},
						},
					},
				},
				Images: []types.ImageWrap{
					{},
					{},
				},
			},
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
				DockerData: &c.dockerData,
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
			countedString := fmt.Sprintf("There are %d", c.counted)
			outOfString := fmt.Sprintf("out of %d", c.outOf)
			assert.True(t, strings.HasPrefix(checkResults[0].Message, countedString))
			assert.True(t, strings.HasSuffix(checkResults[0].Message, outOfString))
		})
	}
}
