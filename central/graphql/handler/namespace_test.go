package handler

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetNamespaces(t *testing.T) {
	mocks := mockResolver(t)
	// DO NOT SUBMIT: fix callers to work with a pointer (go/goprotoapi-findings#message-value)
	loaders.RegisterTypeFactory(reflect.TypeOf(&storage.NamespaceMetadata{}), func() interface{} {
		return loaders.NewNamespaceLoader(mocks.namespace)
	})
	mocks.namespace.EXPECT().Search(gomock.Any(), emptyPaginatedQuery()).Return([]search.Result{
		{
			ID: fakeNamespaceID,
		},
	}, nil)
	nm := &storage.NamespaceMetadata{}
	nm.SetId(fakeNamespaceID)
	nm.SetName(fakeNamespaceName)
	nm.SetClusterId(fakeClusterID)
	nm.SetClusterName(fakeClusterName)
	mocks.namespace.EXPECT().GetManyNamespaces(gomock.Any(), []string{fakeNamespaceID}).Return([]*storage.NamespaceMetadata{
		nm,
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
	nm := &storage.NamespaceMetadata{}
	nm.SetId(fakeNamespaceID)
	nm.SetName(fakeNamespaceName)
	nm.SetClusterId(fakeClusterID)
	nm.SetClusterName(fakeClusterName)
	mocks.namespace.EXPECT().GetNamespace(gomock.Any(), fakeNamespaceID).Return(nm, true, nil)
	response := executeTestQuery(t, mocks, fmt.Sprintf(`{namespace(id:"%s") {metadata{id name clusterId clusterName} }}`, fakeNamespaceID))
	assert.Equal(t, 200, response.Code)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.id", fakeNamespaceID)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.name", fakeNamespaceName)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.clusterId", fakeClusterID)
	assertJSONMatches(t, response.Body, ".data.namespace.metadata.clusterName", fakeClusterName)
}
