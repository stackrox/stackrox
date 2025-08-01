package handler

import (
	"fmt"
	"testing"

	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	deploymentsViewMocks "github.com/stackrox/rox/central/views/deployments/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetDeployment(t *testing.T) {
	mocks := mockResolver(t)
	testClusterID := "testClusterID"
	testDeploymentID := "testDeploymentID"
	mocks.deployment.EXPECT().GetDeployments(gomock.Any(), []string{testDeploymentID}).Return([]*storage.Deployment{
		{Id: testDeploymentID, ClusterId: testClusterID, Name: "deployment name", Type: "deployment type"},
	}, nil)
	mocks.cluster.EXPECT().GetCluster(gomock.Any(), testClusterID).Return(&storage.Cluster{
		Id: testClusterID, Name: "cluster name",
	}, true, nil)

	rec := executeTestQuery(t, mocks, fmt.Sprintf(`{deployment(id: %q){ id name type cluster { name } }}`, testDeploymentID))

	assert.Equal(t, 200, rec.Code)
	assertNoErrors(t, rec.Body)
	assertJSONMatches(t, rec.Body, ".data.deployment.id", testDeploymentID)
	assertJSONMatches(t, rec.Body, ".data.deployment.type", "deployment type")
	assertJSONMatches(t, rec.Body, ".data.deployment.cluster.name", "cluster name")
}

func TestGetDeployments(t *testing.T) {
	t.Setenv(features.FlattenCVEData.EnvVar(), "false")
	if features.FlattenCVEData.Enabled() {
		t.Skip("Flattened CVE data is enabled")
	}

	mocks := mockResolver(t)
	mocks.deployment.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
		{ID: "one"},
		{ID: "two"},
	}, nil)
	mocks.deployment.EXPECT().GetDeployments(gomock.Any(), gomock.Any()).Return([]*storage.Deployment{
		{
			Id: "one", Name: "one name",
		},
		{
			Id: "two", Name: "two name",
		},
	}, nil)

	rec := executeTestQuery(t, mocks, "{deployments { id name }}")

	assert.Equal(t, 200, rec.Code)
	assertNoErrors(t, rec.Body)
	assertJSONMatches(t, rec.Body, ".data.deployments[0].id", "one")
	assertJSONMatches(t, rec.Body, ".data.deployments[1].id", "two")
}

func TestGetDeploymentsFlattenedCVEData(t *testing.T) {
	t.Setenv(features.FlattenCVEData.EnvVar(), "true")
	if !features.FlattenCVEData.Enabled() {
		t.Skip("Flattened CVE data is disabled")
	}

	mocks := mockResolver(t)

	results := make([]deploymentsView.DeploymentCore, 0)
	core1 := deploymentsViewMocks.NewMockDeploymentCore(mocks.ctrl)
	core1.EXPECT().GetDeploymentID().Return("one")
	results = append(results, core1)

	core2 := deploymentsViewMocks.NewMockDeploymentCore(mocks.ctrl)
	core2.EXPECT().GetDeploymentID().Return("two")
	results = append(results, core2)

	mocks.deploymentView.EXPECT().Get(gomock.Any(), gomock.Any()).Return(results, nil)
	mocks.deployment.EXPECT().GetDeployments(gomock.Any(), gomock.Any()).Return([]*storage.Deployment{
		{
			Id: "one", Name: "one name",
		},
		{
			Id: "two", Name: "two name",
		},
	}, nil)

	rec := executeTestQuery(t, mocks, "{deployments { id name }}")

	assert.Equal(t, 200, rec.Code)
	assertNoErrors(t, rec.Body)
	assertJSONMatches(t, rec.Body, ".data.deployments[0].id", "one")
	assertJSONMatches(t, rec.Body, ".data.deployments[1].id", "two")
}

const processQuery = `query d($d:ID) {
	deployment(id:$d) {
		id
		groupedProcesses {
			name timesExecuted groups {
				args signals {  containerName}
			}
    	}
	}
}`

func TestGetDeploymentProcessGroup(t *testing.T) {
	testDeploymentID := "deploymentId"
	mocks := mockResolver(t)
	mocks.deployment.EXPECT().GetDeployments(gomock.Any(), []string{testDeploymentID}).Return([]*storage.Deployment{
		{Id: testDeploymentID},
	}, nil)
	mocks.process.EXPECT().SearchRawProcessIndicators(gomock.Any(), gomock.Any()).Return([]*storage.ProcessIndicator{
		{
			Id:            "processId",
			ContainerName: "container_name",
			DeploymentId:  testDeploymentID,
			PodId:         "podId",
			Signal: &storage.ProcessSignal{
				Id:           "signalId",
				Name:         "process",
				Time:         protocompat.GetProtoTimestampFromSeconds(100),
				ContainerId:  "containerId",
				ExecFilePath: "/bin/process",
				Pid:          1,
				Uid:          0,
				Gid:          0,
			},
		},
	}, nil)
	rec := executeTestQueryWithVariables(t, mocks, processQuery, map[string]string{"d": testDeploymentID})
	assertJSONMatches(t, rec.Body, ".data.deployment.id", testDeploymentID)
	assertJSONMatches(t, rec.Body, ".data.deployment.groupedProcesses[0].name", "/bin/process")
	assertJSONMatches(t, rec.Body, ".data.deployment.groupedProcesses[0].groups[0].signals[0].containerName", "container_name")
}
