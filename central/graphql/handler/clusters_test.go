package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func emptyPaginatedQuery() *v1.Query {
	q := search.EmptyQuery()
	paginated.FillPagination(q, nil, math.MaxInt32)
	return q
}

func TestGetClusters(t *testing.T) {
	mocks := mockResolver(t)
	mocks.cluster.EXPECT().SearchRawClusters(gomock.Any(), emptyPaginatedQuery()).Return([]*storage.Cluster{
		{
			Id:   fakeClusterID,
			Name: "fake cluster",
			Type: storage.ClusterType_KUBERNETES_CLUSTER,
		},
	}, nil)
	response := executeTestQuery(t, mocks, "{clusters {id name type}}")
	assertJSONMatches(t, response.Body, ".data.clusters[0].id", fakeClusterID)
	assert.Equal(t, 200, response.Code)
}

func TestGetCluster(t *testing.T) {
	mocks := mockResolver(t)
	mocks.cluster.EXPECT().GetCluster(gomock.Any(), fakeClusterID).Return(&storage.Cluster{
		Id:   fakeClusterID,
		Name: "fake cluster",
		Type: storage.ClusterType_KUBERNETES_CLUSTER,
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
