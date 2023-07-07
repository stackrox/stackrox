package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/nodecomponent/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	nodeComponent1 = "nodeComponent1"
	nodeComponent2 = "nodeComponent2"
	nodeComponent3 = "nodeComponent3"
)

func TestNodeComponentLoader(t *testing.T) {
	suite.Run(t, new(NodeComponentLoaderTestSuite))
}

type NodeComponentLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *NodeComponentLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *NodeComponentLoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *NodeComponentLoaderTestSuite) TestFromID() {
	loader := nodeComponentLoaderImpl{
		loaded: map[string]*storage.NodeComponent{
			"nodeComponent1": {Id: nodeComponent1},
			"nodeComponent2": {Id: nodeComponent2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded component from id.
	component, err := loader.FromID(suite.ctx, nodeComponent1)
	suite.NoError(err)
	suite.Equal(loader.loaded[nodeComponent1], component)

	// Get a non-preloaded component from id.
	thirdComponent := &storage.NodeComponent{Id: nodeComponent3}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{nodeComponent3}).
		Return([]*storage.NodeComponent{thirdComponent}, nil)

	component, err = loader.FromID(suite.ctx, nodeComponent3)
	suite.NoError(err)
	suite.Equal(thirdComponent, component)

	// Above call should now be preloaded.
	component, err = loader.FromID(suite.ctx, nodeComponent3)
	suite.NoError(err)
	suite.Equal(loader.loaded[nodeComponent3], component)
}

func (suite *NodeComponentLoaderTestSuite) TestFromIDs() {
	loader := nodeComponentLoaderImpl{
		loaded: map[string]*storage.NodeComponent{
			"nodeComponent1": {Id: nodeComponent1},
			"nodeComponent2": {Id: nodeComponent2},
		},
		ds: suite.mockDataStore,
	}

	// Get preloaded components from ids.
	components, err := loader.FromIDs(suite.ctx, []string{nodeComponent1, nodeComponent2})
	suite.NoError(err)
	suite.Equal([]*storage.NodeComponent{
		loader.loaded[nodeComponent1],
		loader.loaded[nodeComponent2],
	}, components)

	// Get a non-preloaded component from id.
	thirdComponent := &storage.NodeComponent{Id: nodeComponent3}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{nodeComponent3}).
		Return([]*storage.NodeComponent{thirdComponent}, nil)

	components, err = loader.FromIDs(suite.ctx, []string{nodeComponent1, nodeComponent2, nodeComponent3})
	suite.NoError(err)
	suite.Equal([]*storage.NodeComponent{
		loader.loaded[nodeComponent1],
		loader.loaded[nodeComponent2],
		thirdComponent,
	}, components)

	// Above call should now be preloaded.
	components, err = loader.FromIDs(suite.ctx, []string{nodeComponent1, nodeComponent2, nodeComponent3})
	suite.NoError(err)
	suite.Equal([]*storage.NodeComponent{
		loader.loaded[nodeComponent1],
		loader.loaded[nodeComponent2],
		loader.loaded[nodeComponent3],
	}, components)
}

func (suite *NodeComponentLoaderTestSuite) TestFromQuery() {
	loader := nodeComponentLoaderImpl{
		loaded: map[string]*storage.NodeComponent{
			"nodeComponent1": {Id: nodeComponent1},
			"nodeComponent2": {Id: nodeComponent2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	results := []search.Result{
		{
			ID: nodeComponent1,
		},
		{
			ID: nodeComponent2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	components, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.NodeComponent{
		loader.loaded[nodeComponent1],
		loader.loaded[nodeComponent2],
	}, components)

	// Get a non-preloaded component
	results = []search.Result{
		{
			ID: nodeComponent1,
		},
		{
			ID: nodeComponent2,
		},
		{
			ID: nodeComponent3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdComponent := &storage.NodeComponent{Id: nodeComponent3}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{nodeComponent3}).
		Return([]*storage.NodeComponent{thirdComponent}, nil)

	components, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.NodeComponent{
		loader.loaded[nodeComponent1],
		loader.loaded[nodeComponent2],
		thirdComponent,
	}, components)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: nodeComponent1,
		},
		{
			ID: nodeComponent2,
		},
		{
			ID: nodeComponent3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	components, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.NodeComponent{
		loader.loaded[nodeComponent1],
		loader.loaded[nodeComponent2],
		loader.loaded[nodeComponent3],
	}, components)
}
