package docker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testStruct struct {
	name      string
	container types.ContainerJSON
	status    framework.Status
}

var (
	indicators = []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  "0",
			ContainerName: "a",
			Signal: &storage.ProcessSignal{
				ContainerId:  "13ea7ce738f4",
				Pid:          15,
				Name:         "ssh",
				ExecFilePath: "/usr/bin/ssh",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  "1",
			ContainerName: "b",
			Signal: &storage.ProcessSignal{
				ContainerId:  "860a6347711e",
				Pid:          32,
				Name:         "sshd",
				ExecFilePath: "/bin/sshd",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  "2",
			ContainerName: "c",
			Signal: &storage.ProcessSignal{
				ContainerId:  "828b7beae96b",
				Pid:          16,
				Name:         "ssh",
				ExecFilePath: "/bin/bash",
			},
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  "3",
			ContainerName: "d",
			Signal: &storage.ProcessSignal{
				ContainerId:  "17e5fdec203e",
				Pid:          33,
				Name:         "sshd",
				ExecFilePath: "/bin/zsh",
			},
		},
	}
)

func TestDockerRuntimeDeploymentChecks(t *testing.T) {
	cases := []*testStruct{
		{
			name: "CIS_Docker_v1_2_0:5_6",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "17e5fdec203e131d823ee0167089847976b9a71f7ad2cafbe45b60ec2bf427b7",
					State: &types.ContainerState{
						Running: true,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_2_0:5_6",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "860a6347711e0989ab0ccdcbe618bcad7cbbb440c27d0d6b02d02388940dd276",
					State: &types.ContainerState{
						Running: true,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_2_0:5_6",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "828b7beae96bb06b275ef589ad6f861e40fc61f7a64c1239526b1ac8df241000",
					State: &types.ContainerState{
						Running: true,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_2_0:5_6",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					ID: "13ea7ce738f4d5921bd2503618a59dfcb48029149d7cbf712063adabbed8e0d2",
					State: &types.ContainerState{
						Running: true,
					},
				},
			},
			status: framework.FailStatus,
		},
	}

	for _, cIt := range cases {
		c := cIt
		t.Run(strings.Replace(c.name, ":", "-", -1), func(t *testing.T) {
			runtimeTest(t, c, checkDeployments)
		})
	}
}

func checkDeployments(t *testing.T, checkResults framework.Results, domain framework.ComplianceDomain, status framework.Status) {
	for _, deployment := range domain.Deployments() {
		nodeResults := checkResults.ForChild(deployment)
		require.NoError(t, nodeResults.Error())
		require.Len(t, nodeResults.Evidence(), 1)
		assert.Equal(t, status, nodeResults.Evidence()[0].Status)
	}
}

func runtimeTest(t *testing.T, c *testStruct, validationFunc func(*testing.T, framework.Results, framework.ComplianceDomain, framework.Status)) {
	t.Parallel()

	registry := framework.RegistrySingleton()
	check := registry.Lookup(c.name)
	require.NotNil(t, check)

	testCluster := &storage.Cluster{
		Id: uuid.NewV4().String(),
	}
	testNodes := createTestNodes("A", "B")

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testPod := createTestPod(&c.container)
	domain := framework.NewComplianceDomain(testCluster, testNodes, nil, testPod, nil)
	data := mocks.NewMockComplianceDataRepository(mockCtrl)

	// Must set the containers to running
	if c.container.ContainerJSONBase != nil {
		if c.container.ContainerJSONBase.State == nil {
			c.container.State = &types.ContainerState{
				Running: true,
			}
		} else {
			c.container.State.Running = true
		}
	} else {
		c.container.ContainerJSONBase = &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: true,
			},
		}
	}
	if c.container.HostConfig == nil {
		c.container.HostConfig = &types.HostConfig{}
	}

	jsonData, err := json.Marshal(&types.Data{
		Containers: []types.ContainerJSON{
			c.container,
		},
	})
	require.NoError(t, err)

	var jsonDataGZ bytes.Buffer
	gzWriter := gzip.NewWriter(&jsonDataGZ)
	_, err = gzWriter.Write(jsonData)
	require.NoError(t, err)
	require.NoError(t, gzWriter.Close())

	data.EXPECT().HostScraped(nodeNameMatcher("A")).AnyTimes().Return(&compliance.ComplianceReturn{
		DockerData: &compliance.GZIPDataChunk{Gzip: jsonDataGZ.Bytes()},
	})
	data.EXPECT().HostScraped(nodeNameMatcher("B")).AnyTimes().Return(&compliance.ComplianceReturn{
		DockerData: &compliance.GZIPDataChunk{Gzip: jsonDataGZ.Bytes()},
	})

	data.EXPECT().SSHProcessIndicators().AnyTimes().Return(indicators)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), "standard", domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[c.name]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 0)
	validationFunc(t, checkResults, domain, c.status)
}

func nodeNameMatcher(nodeName string) gomock.Matcher {
	return testutils.PredMatcher(fmt.Sprintf("node %s", nodeName), func(node *storage.Node) bool {
		return node.GetName() == nodeName
	})
}
