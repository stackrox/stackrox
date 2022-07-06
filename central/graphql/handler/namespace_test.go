package handler

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetNamespaces(t *testing.T) {
	mocks := mockResolver(t)
	mocks.namespace.EXPECT().SearchNamespaces(gomock.Any(), emptyPaginatedQuery()).Return([]*storage.NamespaceMetadata{
		{
			Id:          fakeNamespaceID,
			Name:        fakeNamespaceName,
			ClusterId:   fakeClusterID,
			ClusterName: fakeClusterName,
		},
	}, nil)
	response := executeTestQuery(t, mocks, "{namespaces { metadata { id name clusterId clusterName } } }")
	assert.Equal(t, 200, response.Code)
	assertJSONMatches(t, response.Body, ".data.namespaces[0].metadata.id", fakeNamespaceID)
	assertJSONMatches(t, response.Body, ".data.namespaces[0].metadata.name", fakeNamespaceName)
	assertJSONMatches(t, response.Body, ".data.namespaces[0].metadata.clusterId", fakeClusterID)
	assertJSONMatches(t, response.Body, ".data.namespaces[0].metadata.clusterName", fakeClusterName)
}

func TestGetNamespace(t *testing.T) {
	mocks := mockResolver(t)
	mocks.namespace.EXPECT().GetNamespace(gomock.Any(), fakeNamespaceID).Return(&storage.NamespaceMetadata{
		Id:          fakeNamespaceID,
		Name:        fakeNamespaceName,
		ClusterId:   fakeClusterID,
		ClusterName: fakeClusterName,
	}, true, nil)
	response := executeTestQuery(t, mocks, fmt.Sprintf(`{namespace(id:"%s") {metadata{id name clusterId clusterName} }}`, fakeNamespaceID))
	assert.Equal(t, 200, response.Code)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.id", fakeNamespaceID)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.name", fakeNamespaceName)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.clusterId", fakeClusterID)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.clusterName", fakeClusterName)
}
