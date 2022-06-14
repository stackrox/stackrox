package loaders

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/cve/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

const (
	cve1 = "cve1"
	cve2 = "cve2"
	cve3 = "cve3"
)

func TestCVELoader(t *testing.T) {
	suite.Run(t, new(CVELoaderTestSuite))
}

type CVELoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *CVELoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *CVELoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *CVELoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded cves.
	loader := cveLoaderImpl{
		loaded: map[string]*storage.CVE{
			"cve1": {Id: cve1},
			"cve2": {Id: cve2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cve, err := loader.FromID(suite.ctx, cve1)
	suite.NoError(err)
	suite.Equal(loader.loaded[cve1], cve)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.CVE{Id: cve3}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{cve3}).
		Return([]*storage.CVE{thirdCVE}, nil)

	cve, err = loader.FromID(suite.ctx, cve3)
	suite.NoError(err)
	suite.Equal(thirdCVE, cve)

	// Above call should now be preloaded.
	cve, err = loader.FromID(suite.ctx, cve3)
	suite.NoError(err)
	suite.Equal(loader.loaded[cve3], cve)
}

func (suite *CVELoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded cves.
	loader := cveLoaderImpl{
		loaded: map[string]*storage.CVE{
			"cve1": {Id: cve1},
			"cve2": {Id: cve2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded cve from id.
	cves, err := loader.FromIDs(suite.ctx, []string{cve1, cve2})
	suite.NoError(err)
	suite.Equal([]*storage.CVE{
		loader.loaded[cve1],
		loader.loaded[cve2],
	}, cves)

	// Get a non-preloaded cve from id.
	thirdCVE := &storage.CVE{Id: "cve3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{cve3}).
		Return([]*storage.CVE{thirdCVE}, nil)

	cves, err = loader.FromIDs(suite.ctx, []string{cve1, cve2, cve3})
	suite.NoError(err)
	suite.Equal([]*storage.CVE{
		loader.loaded[cve1],
		loader.loaded[cve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	cves, err = loader.FromIDs(suite.ctx, []string{cve1, cve2, cve3})
	suite.NoError(err)
	suite.Equal([]*storage.CVE{
		loader.loaded[cve1],
		loader.loaded[cve2],
		loader.loaded[cve3],
	}, cves)
}

func (suite *CVELoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded cves.
	loader := cveLoaderImpl{
		loaded: map[string]*storage.CVE{
			"cve1": {Id: cve1},
			"cve2": {Id: cve2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded cve from id.
	results := []search.Result{
		{
			ID: cve1,
		},
		{
			ID: cve2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.CVE{
		loader.loaded[cve1],
		loader.loaded[cve2],
	}, cves)

	// Get a non-preloaded cve from id.
	results = []search.Result{
		{
			ID: cve1,
		},
		{
			ID: cve2,
		},
		{
			ID: cve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdCVE := &storage.CVE{Id: "cve3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{cve3}).
		Return([]*storage.CVE{thirdCVE}, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.CVE{
		loader.loaded[cve1],
		loader.loaded[cve2],
		thirdCVE,
	}, cves)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: cve1,
		},
		{
			ID: cve2,
		},
		{
			ID: cve3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	cves, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.CVE{
		loader.loaded[cve1],
		loader.loaded[cve2],
		loader.loaded[cve3],
	}, cves)
}
