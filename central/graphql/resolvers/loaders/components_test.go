package loaders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	component1 = "component1"
	component2 = "component2"
	component3 = "component3"
)

func TestComponentLoader(t *testing.T) {
	suite.Run(t, new(ComponentLoaderTestSuite))
}

type ComponentLoaderTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl      *gomock.Controller
	mockDataStore *mocks.MockDataStore
}

func (suite *ComponentLoaderTestSuite) SetupTest() {
	suite.ctx = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDataStore = mocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *ComponentLoaderTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ComponentLoaderTestSuite) TestFromID() {
	// Create a loader with some reloaded components.
	loader := componentLoaderImpl{
		loaded: map[string]*storage.ImageComponent{
			"component1": {Id: component1},
			"component2": {Id: component2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded component from id.
	component, err := loader.FromID(suite.ctx, component1)
	suite.NoError(err)
	suite.Equal(loader.loaded[component1], component)

	// Get a non-preloaded component from id.
	thirdComponent := &storage.ImageComponent{Id: component3}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{component3}).
		Return([]*storage.ImageComponent{thirdComponent}, nil)

	component, err = loader.FromID(suite.ctx, component3)
	suite.NoError(err)
	suite.Equal(thirdComponent, component)

	// Above call should now be preloaded.
	component, err = loader.FromID(suite.ctx, component3)
	suite.NoError(err)
	suite.Equal(loader.loaded[component3], component)
}

func (suite *ComponentLoaderTestSuite) TestFromIDs() {
	// Create a loader with some reloaded components.
	loader := componentLoaderImpl{
		loaded: map[string]*storage.ImageComponent{
			"component1": {Id: component1},
			"component2": {Id: component2},
		},
		ds: suite.mockDataStore,
	}

	// Get a preloaded component from id.
	components, err := loader.FromIDs(suite.ctx, []string{component1, component2})
	suite.NoError(err)
	suite.Equal([]*storage.ImageComponent{
		loader.loaded[component1],
		loader.loaded[component2],
	}, components)

	// Get a non-preloaded component from id.
	thirdComponent := &storage.ImageComponent{Id: "component3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{component3}).
		Return([]*storage.ImageComponent{thirdComponent}, nil)

	components, err = loader.FromIDs(suite.ctx, []string{component1, component2, component3})
	suite.NoError(err)
	suite.Equal([]*storage.ImageComponent{
		loader.loaded[component1],
		loader.loaded[component2],
		thirdComponent,
	}, components)

	// Above call should now be preloaded.
	components, err = loader.FromIDs(suite.ctx, []string{component1, component2, component3})
	suite.NoError(err)
	suite.Equal([]*storage.ImageComponent{
		loader.loaded[component1],
		loader.loaded[component2],
		loader.loaded[component3],
	}, components)
}

func (suite *ComponentLoaderTestSuite) TestFromQuery() {
	// Create a loader with some reloaded components.
	loader := componentLoaderImpl{
		loaded: map[string]*storage.ImageComponent{
			"component1": {Id: component1},
			"component2": {Id: component2},
		},
		ds: suite.mockDataStore,
	}
	query := &v1.Query{}

	// Get a preloaded component from id.
	results := []search.Result{
		{
			ID: component1,
		},
		{
			ID: component2,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	components, err := loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ImageComponent{
		loader.loaded[component1],
		loader.loaded[component2],
	}, components)

	// Get a non-preloaded component from id.
	results = []search.Result{
		{
			ID: component1,
		},
		{
			ID: component2,
		},
		{
			ID: component3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	thirdComponent := &storage.ImageComponent{Id: "component3"}
	suite.mockDataStore.EXPECT().GetBatch(suite.ctx, []string{component3}).
		Return([]*storage.ImageComponent{thirdComponent}, nil)

	components, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ImageComponent{
		loader.loaded[component1],
		loader.loaded[component2],
		thirdComponent,
	}, components)

	// Above call should now be preloaded.
	results = []search.Result{
		{
			ID: component1,
		},
		{
			ID: component2,
		},
		{
			ID: component3,
		},
	}
	suite.mockDataStore.EXPECT().Search(suite.ctx, query).Return(results, nil)

	components, err = loader.FromQuery(suite.ctx, query)
	suite.NoError(err)
	suite.Equal([]*storage.ImageComponent{
		loader.loaded[component1],
		loader.loaded[component2],
		loader.loaded[component3],
	}, components)
}
