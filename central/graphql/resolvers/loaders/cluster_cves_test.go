package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	clusterCve1 = "cve1"
	clusterCve2 = "cve2"
	clusterCve3 = "cve3"
)

func TestClusterCVELoader(t *testing.T) {
	suite.Run(t, new(ClusterCVELoaderTestSuite))
}

type ClusterCVELoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *ClusterCVELoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *ClusterCVELoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ClusterCVELoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded cves.
	loader := clusterCveLoaderImpl{
		loaded: map[string]*storage.ClusterCVE{
			"cve1": {Id: clusterCve1},
			"cve2": {Id: clusterCve2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cve, err := loader.FromID(suite.ctx, clusterCve1)
	suite.NoError(err)
	suite.Equal(loader.loaded[clusterCve1], cve)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.ClusterCVE{Id: clusterCve3}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{clusterCve3}).
		Return([]*storage.ClusterCVE{thirdCVE}, nil)

	cve, err = loader.FromID(suite.ctx, clusterCve3)
	suite.NoError(err)
	suite.Equal(thirdCVE, cve)

	// Above call should now be preloaded.
	cve, err = loader.FromID(suite.ctx, clusterCve3)
	suite.NoError(err)
	suite.Equal(loader.loaded[clusterCve3], cve)
}

func (suite *ClusterCVELoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded cves.
	loader := clusterCveLoaderImpl{
		loaded: map[string]*storage.ClusterCVE{
			"cve1": {Id: clusterCve1},
			"cve2": {Id: clusterCve2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cves, err := loader.FromIDs(suite.ctx, []string{clusterCve1, clusterCve2})
	suite.NoError(err)
	suite.Equal([]*storage.ClusterCVE{
		loader.loaded[clusterCve1],
		loader.loaded[clusterCve2],
	}, cves)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.ClusterCVE{Id: "cve3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{clusterCve3}).
		Return([]*storage.ClusterCVE{thirdCVE}, nil)

	cves, err = loader.FromIDs(suite.ctx, []string{clusterCve1, clusterCve2, clusterCve3})
	suite.NoError(err)
	suite.Equal([]*storage.ClusterCVE{
		loader.loaded[clusterCve1],
		loader.loaded[clusterCve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	cves, err = loader.FromIDs(suite.ctx, []string{clusterCve1, clusterCve2, clusterCve3})
	suite.NoError(err)
	suite.Equal([]*storage.ClusterCVE{
		loader.loaded[clusterCve1],
		loader.loaded[clusterCve2],
		loader.loaded[clusterCve3],
	}, cves)
}

func (suite *ClusterCVELoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded cves.
	loader := clusterCveLoaderImpl{
		loaded: map[string]*storage.ClusterCVE{
			"cve1": {Id: clusterCve1},
			"cve2": {Id: clusterCve2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded cve from id.
	results := []search.Result{
		{
			ID: clusterCve1,
		},
		{
			ID: clusterCve2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ClusterCVE{
		loader.loaded[clusterCve1],
		loader.loaded[clusterCve2],
	}, cves)

	// Get a non-preloaded cve from id.
	results = []search.Result{
		{
			ID: clusterCve1,
		},
		{
			ID: clusterCve2,
		},
		{
			ID: clusterCve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdCVE := &storage.ClusterCVE{Id: "cve3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{clusterCve3}).
		Return([]*storage.ClusterCVE{thirdCVE}, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ClusterCVE{
		loader.loaded[clusterCve1],
		loader.loaded[clusterCve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: clusterCve1,
		},
		{
			ID: clusterCve2,
		},
		{
			ID: clusterCve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ClusterCVE{
		loader.loaded[clusterCve1],
		loader.loaded[clusterCve2],
		loader.loaded[clusterCve3],
	}, cves)
}
