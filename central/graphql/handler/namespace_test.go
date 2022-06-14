package handler

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
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
	mocks.deployment.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
		{
			ID: fakeDeploymentID,
		},
	}, nil)
	mocks.secret.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
		{
			ID: fakeSecretID,
		},
	}, nil)
	mocks.nps.EXPECT().CountMatchingNetworkPolicies(gomock.Any(), fakeClusterID, fakeNamespaceName).Return(1, nil)
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
	mocks.deployment.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
		{
			ID: fakeDeploymentID,
		},
	}, nil)
	mocks.secret.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
		{
			ID: fakeSecretID,
		},
	}, nil)
	mocks.nps.EXPECT().CountMatchingNetworkPolicies(gomock.Any(), fakeClusterID, fakeNamespaceName).Return(1, nil)
	response := executeTestQuery(t, mocks, fmt.Sprintf(`{namespace(id:"%s") {metadata{id name clusterId clusterName} }}`, fakeNamespaceID))
	assert.Equal(t, 200, response.Code)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.id", fakeNamespaceID)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.name", fakeNamespaceName)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.clusterId", fakeClusterID)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.clusterName", fakeClusterName)
}
