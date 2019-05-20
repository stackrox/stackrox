package service

import (
	"context"
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/processwhitelist/index"
	whitelistSearch "github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	"github.com/stackrox/rox/central/reprocessor/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func fillDB(t *testing.T, ds datastore.DataStore, whitelists []*storage.ProcessWhitelist) {
	initialContents, err := ds.GetProcessWhitelists(context.TODO())
	assert.NoError(t, err)
	assert.Empty(t, initialContents, "initial db not empty, you have a test bug")
	for _, whitelist := range whitelists {
		_, err := ds.AddProcessWhitelist(context.TODO(), whitelist)
		assert.NoError(t, err)
	}
}

func emptyDB(t *testing.T, ds datastore.DataStore) {
	whitelists, err := ds.GetProcessWhitelists(context.TODO())
	assert.NoError(t, err)
	for _, whitelist := range whitelists {
		assert.NoError(t, ds.RemoveProcessWhitelist(context.TODO(), whitelist.GetKey()))
	}
}

func TestProcessWhitelistService(t *testing.T) {
	suite.Run(t, new(ProcessWhitelistServiceTestSuite))
}

type ProcessWhitelistServiceTestSuite struct {
	suite.Suite
	datastore   datastore.DataStore
	service     Service
	db          *bbolt.DB
	reprocessor *mocks.MockLoop
	mockCtrl    *gomock.Controller
}

func (suite *ProcessWhitelistServiceTestSuite) SetupTest() {
	var err error
	suite.db, err = bolthelper.NewTemp("process_whitelist_service_test.db")
	suite.NoError(err)
	wlStore, err := store.New(suite.db)
	suite.NoError(err)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.NoError(err)
	indexer := index.New(tmpIndex)

	searcher, err := whitelistSearch.New(wlStore, indexer)
	suite.NoError(err)

	suite.datastore = datastore.New(wlStore, indexer, searcher)
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.reprocessor = mocks.NewMockLoop(suite.mockCtrl)
	suite.service = New(suite.datastore, suite.reprocessor)
}

func (suite *ProcessWhitelistServiceTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
	suite.mockCtrl.Finish()
}

func (suite *ProcessWhitelistServiceTestSuite) TestGetProcessWhitelists() {
	cases := []struct {
		name       string
		whitelists []*storage.ProcessWhitelist
	}{
		{
			name: "Empty DB",
		},
		{
			name:       "One whitelist",
			whitelists: []*storage.ProcessWhitelist{fixtures.GetProcessWhitelistWithKey()},
		},
		{
			name: "Many whitelists",
			whitelists: []*storage.ProcessWhitelist{
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
			},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			fillDB(t, suite.datastore, c.whitelists)
			defer emptyDB(t, suite.datastore)
			gotWhitelists, err := suite.service.GetProcessWhitelists((context.Context)(nil), nil)
			assert.NoError(t, err)
			assert.ElementsMatch(t, gotWhitelists.Whitelists, c.whitelists)
		})
	}
}

func (suite *ProcessWhitelistServiceTestSuite) TestGetProcessWhitelist() {
	knownWhitelist := fixtures.GetProcessWhitelistWithKey()
	cases := []struct {
		name           string
		whitelists     []*storage.ProcessWhitelist
		expectedResult *storage.ProcessWhitelist
		shouldFail     bool
	}{
		{
			name:       "Empty DB",
			whitelists: []*storage.ProcessWhitelist{},
			shouldFail: true,
		},
		{
			name:           "One whitelist",
			whitelists:     []*storage.ProcessWhitelist{knownWhitelist},
			expectedResult: knownWhitelist,
			shouldFail:     false,
		},
		{
			name: "Many Whitelists",
			whitelists: []*storage.ProcessWhitelist{
				knownWhitelist,
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
			},
			expectedResult: knownWhitelist,
			shouldFail:     false,
		},
		{
			name: "Search for non-existant",
			whitelists: []*storage.ProcessWhitelist{
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
			},
			shouldFail: true,
		},
	}
	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			fillDB(t, suite.datastore, c.whitelists)
			defer emptyDB(t, suite.datastore)
			requestByKey := &v1.GetProcessWhitelistRequest{Key: knownWhitelist.GetKey()}
			whitelist, err := suite.service.GetProcessWhitelist((context.Context)(nil), requestByKey)
			if c.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedResult, whitelist)
			}
		})
	}
}

func (suite *ProcessWhitelistServiceTestSuite) TestUpdateProcessWhitelist() {
	stockProcesses := []string{"stock_process_1", "stock_process_2"}

	whitelistCollection := make(map[int]*storage.ProcessWhitelist)
	getWhitelist := func(index int) *storage.ProcessWhitelist {
		if whitelist, ok := whitelistCollection[index]; ok {
			return whitelist
		}
		whitelist := fixtures.GetProcessWhitelistWithKey()
		whitelist.Elements = make([]*storage.WhitelistElement, 0, len(stockProcesses))
		for _, stockProcess := range stockProcesses {
			whitelist.Elements = append(whitelist.Elements, &storage.WhitelistElement{
				Element: &storage.WhitelistItem{Item: &storage.WhitelistItem_ProcessName{ProcessName: stockProcess}},
			})
		}
		whitelistCollection[index] = whitelist
		return whitelist
	}

	getWhitelists := func(indexes ...int) []*storage.ProcessWhitelist {
		whitelists := make([]*storage.ProcessWhitelist, 0, len(indexes))
		for _, i := range indexes {
			whitelists = append(whitelists, getWhitelist(i))
		}
		return whitelists
	}

	getWhitelistKey := func(index int) *storage.ProcessWhitelistKey {
		return getWhitelist(index).GetKey()
	}

	getWhitelistKeys := func(indexes ...int) []*storage.ProcessWhitelistKey {
		keys := make([]*storage.ProcessWhitelistKey, 0, len(indexes))
		for _, i := range indexes {
			keys = append(keys, getWhitelistKey(i))
		}
		return keys
	}

	cases := []struct {
		name                string
		whitelists          []*storage.ProcessWhitelist
		toUpdate            []*storage.ProcessWhitelistKey
		toAdd               []string
		toRemove            []string
		expectedSuccessKeys []*storage.ProcessWhitelistKey
		expectedErrorKeys   []*storage.ProcessWhitelistKey
	}{
		{
			name:              "Update non-existent",
			toUpdate:          getWhitelistKeys(0, 1),
			toAdd:             []string{"Doesn't matter"},
			toRemove:          []string{"whatever"},
			expectedErrorKeys: getWhitelistKeys(0, 1),
		},
		{
			name:                "Update one",
			whitelists:          getWhitelists(0),
			toUpdate:            getWhitelistKeys(0),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[0]},
			expectedSuccessKeys: getWhitelistKeys(0),
		},
		{
			name:                "Update many",
			whitelists:          getWhitelists(0, 1, 2, 3, 4),
			toUpdate:            getWhitelistKeys(0, 1, 2, 3, 4),
			toAdd:               []string{"Some process"},
			expectedSuccessKeys: getWhitelistKeys(0, 1, 2, 3, 4),
		},
		{
			name:                "Mixed failures",
			whitelists:          getWhitelists(0),
			toUpdate:            getWhitelistKeys(0, 1),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[0]},
			expectedSuccessKeys: getWhitelistKeys(0),
			expectedErrorKeys:   getWhitelistKeys(1),
		},
		{
			name:                "Unrelated list",
			whitelists:          getWhitelists(0, 1),
			toUpdate:            getWhitelistKeys(0),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[1]},
			expectedSuccessKeys: getWhitelistKeys(0),
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			fillDB(t, suite.datastore, c.whitelists)
			defer emptyDB(t, suite.datastore)

			request := &v1.UpdateProcessWhitelistsRequest{
				Keys:           c.toUpdate,
				AddElements:    fixtures.MakeWhitelistItems(c.toAdd...),
				RemoveElements: fixtures.MakeWhitelistItems(c.toRemove...),
			}
			suite.reprocessor.EXPECT().ReprocessRiskForDeployments(gomock.Any())
			response, err := suite.service.UpdateProcessWhitelists((context.Context)(nil), request)
			assert.NoError(t, err)
			var successKeys []*storage.ProcessWhitelistKey
			for _, wl := range response.Whitelists {
				successKeys = append(successKeys, wl.GetKey())
				processes := set.NewStringSet()
				for _, process := range wl.Elements {
					processes.Add(process.GetElement().GetProcessName())
				}
				for _, add := range c.toAdd {
					assert.True(t, processes.Contains(add))
				}
				for _, remove := range c.toRemove {
					assert.False(t, processes.Contains(remove))
				}
				for _, stockProcess := range stockProcesses {
					if sliceutils.StringFind(c.toRemove, stockProcess) == -1 {
						assert.True(t, processes.Contains(stockProcess))
					}
				}
			}
			assert.ElementsMatch(t, c.expectedSuccessKeys, successKeys)
			var errorKeys []*storage.ProcessWhitelistKey
			for _, err := range response.Errors {
				errorKeys = append(errorKeys, err.GetKey())
			}
			assert.ElementsMatch(t, c.expectedErrorKeys, errorKeys)
		})
	}
}
