package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/namespace/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	namespace1 = "namespace1"
	namespace2 = "namespace2"
	namespace3 = "namespace3"
)

func TestNamespaceLoader(t *testing.T) {
	suite.Run(t, new(NamespaceLoaderTestSuite))
}

type NamespaceLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *NamespaceLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *NamespaceLoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *NamespaceLoaderTestSuite) TestFromID() {
	loader := namespaceLoaderImpl{
		loaded: map[string]*storage.NamespaceMetadata{
			"namespace1": {Id: namespace1},
			"namespace2": {Id: namespace2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded namespace from id.
	namespace, err := loader.FromID(suite.ctx, namespace1)
	suite.NoError(err)
	suite.Equal(loader.loaded[namespace1], namespace)

	// Get a non-preloaded namespace from id.
	thirdNamespace := &storage.NamespaceMetadata{Id: namespace3}
	suite.mockDataStore.EXPECT().GetManyNamespaces(suite.ctx, []string{namespace3}).
		Return([]*storage.NamespaceMetadata{thirdNamespace}, nil)

	namespace, err = loader.FromID(suite.ctx, namespace3)
	suite.NoError(err)
	suite.Equal(thirdNamespace, namespace)

	// Above call should now be preloaded.
	namespace, err = loader.FromID(suite.ctx, namespace3)
	suite.NoError(err)
	suite.Equal(loader.loaded[namespace3], namespace)
}

func (suite *NamespaceLoaderTestSuite) TestFromIDs() {
	loader := namespaceLoaderImpl{
		loaded: map[string]*storage.NamespaceMetadata{
			"namespace1": {Id: namespace1},
			"namespace2": {Id: namespace2},
		},
		ds: suite.mockDataStore,
	}

	// Get preloaded namespaces from ids.
	namespaces, err := loader.FromIDs(suite.ctx, []string{namespace1, namespace2})
	suite.NoError(err)
	suite.Equal([]*storage.NamespaceMetadata{
		loader.loaded[namespace1],
		loader.loaded[namespace2],
	}, namespaces)

	// Get a non-preloaded namespace from id.
	thirdNamespace := &storage.NamespaceMetadata{Id: namespace3}
	suite.mockDataStore.EXPECT().GetManyNamespaces(suite.ctx, []string{namespace3}).
		Return([]*storage.NamespaceMetadata{thirdNamespace}, nil)

	namespaces, err = loader.FromIDs(suite.ctx, []string{namespace1, namespace2, namespace3})
	suite.NoError(err)
	suite.Equal([]*storage.NamespaceMetadata{
		loader.loaded[namespace1],
		loader.loaded[namespace2],
		thirdNamespace,
	}, namespaces)

	// Above call should now be preloaded.
	namespaces, err = loader.FromIDs(suite.ctx, []string{namespace1, namespace2, namespace3})
	suite.NoError(err)
	suite.Equal([]*storage.NamespaceMetadata{
		loader.loaded[namespace1],
		loader.loaded[namespace2],
		loader.loaded[namespace3],
	}, namespaces)
}

func (suite *NamespaceLoaderTestSuite) TestFromQuery() {
	loader := namespaceLoaderImpl{
		loaded: map[string]*storage.NamespaceMetadata{
			"namespace1": {Id: namespace1},
			"namespace2": {Id: namespace2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	results := []search.Result{
		{
			ID: namespace1,
		},
		{
			ID: namespace2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	namespaces, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.NamespaceMetadata{
		loader.loaded[namespace1],
		loader.loaded[namespace2],
	}, namespaces)

	// Get a non-preloaded namespace
	results = []search.Result{
		{
			ID: namespace1,
		},
		{
			ID: namespace2,
		},
		{
			ID: namespace3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdNamespace := &storage.NamespaceMetadata{Id: namespace3}
	suite.mockDataStore.EXPECT().GetManyNamespaces(suite.ctx, []string{namespace3}).
		Return([]*storage.NamespaceMetadata{thirdNamespace}, nil)

	namespaces, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.NamespaceMetadata{
		loader.loaded[namespace1],
		loader.loaded[namespace2],
		thirdNamespace,
	}, namespaces)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: namespace1,
		},
		{
			ID: namespace2,
		},
		{
			ID: namespace3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	namespaces, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.NamespaceMetadata{
		loader.loaded[namespace1],
		loader.loaded[namespace2],
		loader.loaded[namespace3],
	}, namespaces)
}
