package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/node/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	nodeCve1 = "nodeCve1"
	nodeCve2 = "nodeCve2"
	nodeCve3 = "nodeCve3"
)

func TestNodeCVELoader(t *testing.T) {
	suite.Run(t, new(NodeCVELoaderTestSuite))
}

type NodeCVELoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *NodeCVELoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *NodeCVELoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *NodeCVELoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded cves.
	nodeCVE := &storage.NodeCVE{}
	nodeCVE.SetId(nodeCve1)
	nodeCVE2 := &storage.NodeCVE{}
	nodeCVE2.SetId(nodeCve2)
	loader := nodeCVELoaderImpl{
		loaded: map[string]*storage.NodeCVE{
			"nodeCve1": nodeCVE,
			"nodeCve2": nodeCVE2,
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cve, err := loader.FromID(suite.ctx, nodeCve1)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[nodeCve1], cve)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.NodeCVE{}
	thirdCVE.SetId(nodeCve3)
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{nodeCve3}).
		Return([]*storage.NodeCVE{thirdCVE}, nil)

	cve, err = loader.FromID(suite.ctx, nodeCve3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), thirdCVE, cve)

	// Above call should now be preloaded.
	cve, err = loader.FromID(suite.ctx, nodeCve3)
	suite.NoError(err)
	protoassert.Equal(suite.T(), loader.loaded[nodeCve3], cve)
}

func (suite *NodeCVELoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded cves.
	nodeCVE := &storage.NodeCVE{}
	nodeCVE.SetId(nodeCve1)
	nodeCVE2 := &storage.NodeCVE{}
	nodeCVE2.SetId(nodeCve2)
	loader := nodeCVELoaderImpl{
		loaded: map[string]*storage.NodeCVE{
			"nodeCve1": nodeCVE,
			"nodeCve2": nodeCVE2,
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cves, err := loader.FromIDs(suite.ctx, []string{nodeCve1, nodeCve2})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.NodeCVE{
		loader.loaded[nodeCve1],
		loader.loaded[nodeCve2],
	}, cves)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.NodeCVE{}
	thirdCVE.SetId("nodeCve3")
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{nodeCve3}).
		Return([]*storage.NodeCVE{thirdCVE}, nil)

	cves, err = loader.FromIDs(suite.ctx, []string{nodeCve1, nodeCve2, nodeCve3})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.NodeCVE{
		loader.loaded[nodeCve1],
		loader.loaded[nodeCve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	cves, err = loader.FromIDs(suite.ctx, []string{nodeCve1, nodeCve2, nodeCve3})
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.NodeCVE{
		loader.loaded[nodeCve1],
		loader.loaded[nodeCve2],
		loader.loaded[nodeCve3],
	}, cves)
}

func (suite *NodeCVELoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded cves.
	nodeCVE := &storage.NodeCVE{}
	nodeCVE.SetId(nodeCve1)
	nodeCVE2 := &storage.NodeCVE{}
	nodeCVE2.SetId(nodeCve2)
	loader := nodeCVELoaderImpl{
		loaded: map[string]*storage.NodeCVE{
			"nodeCve1": nodeCVE,
			"nodeCve2": nodeCVE2,
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded cve from id.
	results := []search.Result{
		{
			ID: nodeCve1,
		},
		{
			ID: nodeCve2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.NodeCVE{
		loader.loaded[nodeCve1],
		loader.loaded[nodeCve2],
	}, cves)

	// Get a non-preloaded cve from id.
	results = []search.Result{
		{
			ID: nodeCve1,
		},
		{
			ID: nodeCve2,
		},
		{
			ID: nodeCve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdCVE := &storage.NodeCVE{}
	thirdCVE.SetId("nodeCve3")
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{nodeCve3}).
		Return([]*storage.NodeCVE{thirdCVE}, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.NodeCVE{
		loader.loaded[nodeCve1],
		loader.loaded[nodeCve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: nodeCve1,
		},
		{
			ID: nodeCve2,
		},
		{
			ID: nodeCve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	protoassert.SlicesEqual(suite.T(), []*storage.NodeCVE{
		loader.loaded[nodeCve1],
		loader.loaded[nodeCve2],
		loader.loaded[nodeCve3],
	}, cves)
}
