package handler

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetClusters(t *testing.T) {
	mocks := mockResolver(t)
	mocks.cluster.EXPECT().GetClusters().Return([]*v1.Cluster{
		{
			Id:   fakeClusterID,
			Name: "fake cluster",
			Type: v1.ClusterType_KUBERNETES_CLUSTER,
			OrchestratorParams: &v1.Cluster_Kubernetes{
				Kubernetes: &v1.KubernetesParams{
					Params: &v1.CommonKubernetesParams{Namespace: "stackrox"},
				},
			},
		},
	}, nil)
	response := executeTestQuery(t, mocks, "{clusters {id name type orchestratorParams {... on KubernetesParams { params {namespace}}}}}")
	assertJSONMatches(t, response.Body, ".data.clusters[0].id", fakeClusterID)
	assertJSONMatches(t, response.Body, ".data.clusters[0].orchestratorParams.params.namespace", "stackrox")
	assert.Equal(t, 200, response.Code)
}

func TestGetCluster(t *testing.T) {
	mocks := mockResolver(t)
	mocks.cluster.EXPECT().GetCluster(fakeClusterID).Return(&v1.Cluster{
		Id:   fakeClusterID,
		Name: "fake cluster",
		Type: v1.ClusterType_KUBERNETES_CLUSTER,
	}, true, nil)
	response := executeTestQuery(t, mocks, fmt.Sprintf(`{cluster(id: "%s") { id name type}}`, fakeClusterID))
	assert.Equal(t, 200, response.Code)
	assertJSONMatches(t, response.Body, ".data.cluster.id", fakeClusterID)
	j := map[string]*json.RawMessage{}
	err := json.Unmarshal(response.Body.Bytes(), &j)
	if err != nil {
		t.Fatal(err)
	}
}
