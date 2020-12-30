package service

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/processbaseline/index"
	whitelistSearch "github.com/stackrox/rox/central/processbaseline/search"
	rocksdbStore "github.com/stackrox/rox/central/processbaseline/store/rocksdb"
	resultsMocks "github.com/stackrox/rox/central/processbaselineresults/datastore/mocks"
	"github.com/stackrox/rox/central/reprocessor/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TODO(ROX-6194): Remove this file after the deprecation cycle started with the 54.0 release.

func TestProcessWhitelistService(t *testing.T) {
	suite.Run(t, new(ProcessWhitelistServiceTestSuite))
}

type ProcessWhitelistServiceTestSuite struct {
	suite.Suite
	datastore datastore.DataStore
	service   Service

	db *rocksdb.RocksDB

	reprocessor     *mocks.MockLoop
	resultDatastore *resultsMocks.MockDataStore
	connectionMgr   *connectionMocks.MockManager
	mockCtrl        *gomock.Controller
}

func (suite *ProcessWhitelistServiceTestSuite) SetupTest() {
	db, err := rocksdb.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err)

	suite.db = db

	store, err := rocksdbStore.New(db)
	suite.NoError(err)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.NoError(err)
	indexer := index.New(tmpIndex)

	searcher, err := whitelistSearch.New(store, indexer)
	suite.NoError(err)

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.resultDatastore = resultsMocks.NewMockDataStore(suite.mockCtrl)
	suite.resultDatastore.EXPECT().DeleteBaselineResults(gomock.Any(), gomock.Any()).AnyTimes()

	suite.datastore = datastore.New(store, indexer, searcher, suite.resultDatastore)
	suite.reprocessor = mocks.NewMockLoop(suite.mockCtrl)
	suite.connectionMgr = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.service = New(suite.datastore, suite.reprocessor, suite.connectionMgr)
}

func (suite *ProcessWhitelistServiceTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
	suite.mockCtrl.Finish()
}

func (suite *ProcessWhitelistServiceTestSuite) TestGetProcessWhitelist() {
	knownWhitelist := fixtures.GetProcessBaselineWithKey()
	cases := []struct {
		name           string
		whitelists     []*storage.ProcessBaseline
		expectedResult *storage.ProcessBaseline
		shouldFail     bool
	}{
		{
			name:       "Empty DB",
			whitelists: []*storage.ProcessBaseline{},
			shouldFail: true,
		},
		{
			name:           "One process baseline",
			whitelists:     []*storage.ProcessBaseline{knownWhitelist},
			expectedResult: knownWhitelist,
			shouldFail:     false,
		},
		{
			name: "Many process baselines",
			whitelists: []*storage.ProcessBaseline{
				knownWhitelist,
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
			},
			expectedResult: knownWhitelist,
			shouldFail:     false,
		},
		{
			name: "Search for non-existent",
			whitelists: []*storage.ProcessBaseline{
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
			},
			shouldFail: true,
		},
	}
	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			fillDB(t, suite.datastore, c.whitelists)
			defer emptyDB(t, suite.datastore, c.whitelists)
			requestByKey := &v1.GetProcessWhitelistRequest{Key: knownWhitelist.GetKey()}
			whitelist, err := suite.service.GetProcessWhitelist(hasReadCtx, requestByKey)
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

	whitelistCollection := make(map[int]*storage.ProcessBaseline)
	getWhitelist := func(index int) *storage.ProcessBaseline {
		if whitelist, ok := whitelistCollection[index]; ok {
			return whitelist
		}
		whitelist := fixtures.GetProcessBaselineWithKey()
		whitelist.Elements = make([]*storage.BaselineElement, 0, len(stockProcesses))
		for _, stockProcess := range stockProcesses {
			whitelist.Elements = append(whitelist.Elements, &storage.BaselineElement{
				Element: &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: stockProcess}},
			})
		}
		whitelistCollection[index] = whitelist
		return whitelist
	}

	getWhitelists := func(indexes ...int) []*storage.ProcessBaseline {
		whitelists := make([]*storage.ProcessBaseline, 0, len(indexes))
		for _, i := range indexes {
			whitelists = append(whitelists, getWhitelist(i))
		}
		return whitelists
	}

	getWhitelistKey := func(index int) *storage.ProcessBaselineKey {
		return getWhitelist(index).GetKey()
	}

	getWhitelistKeys := func(indexes ...int) []*storage.ProcessBaselineKey {
		keys := make([]*storage.ProcessBaselineKey, 0, len(indexes))
		for _, i := range indexes {
			keys = append(keys, getWhitelistKey(i))
		}
		return keys
	}

	cases := []struct {
		name                string
		whitelists          []*storage.ProcessBaseline
		toUpdate            []*storage.ProcessBaselineKey
		toAdd               []string
		toRemove            []string
		expectedSuccessKeys []*storage.ProcessBaselineKey
		expectedErrorKeys   []*storage.ProcessBaselineKey
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
			defer emptyDB(t, suite.datastore, c.whitelists)

			request := &v1.UpdateProcessWhitelistsRequest{
				Keys:           c.toUpdate,
				AddElements:    fixtures.MakeBaselineItems(c.toAdd...),
				RemoveElements: fixtures.MakeBaselineItems(c.toRemove...),
			}
			suite.reprocessor.EXPECT().ReprocessRiskForDeployments(gomock.Any())
			for range c.expectedSuccessKeys {
				suite.connectionMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any())
			}
			response, err := suite.service.UpdateProcessWhitelists(hasWriteCtx, request)
			assert.NoError(t, err)
			var successKeys []*storage.ProcessBaselineKey
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
			var errorKeys []*storage.ProcessBaselineKey
			for _, err := range response.Errors {
				errorKeys = append(errorKeys, err.GetKey())
			}
			assert.ElementsMatch(t, c.expectedErrorKeys, errorKeys)
		})
	}
}

func (suite *ProcessWhitelistServiceTestSuite) TestDeleteProcessWhitelists() {
	whitelists := []*storage.ProcessBaseline{
		{
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  "d1",
				ContainerName: "container",
				ClusterId:     "clusterid",
				Namespace:     "namespace",
			},
		},
		{
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  "d2",
				ContainerName: "container",
				ClusterId:     "clusterid",
				Namespace:     "namespace",
			},
		},
	}

	for _, whitelist := range whitelists {
		id, err := suite.datastore.AddProcessBaseline(hasWriteCtx, whitelist)
		suite.NoError(err)
		whitelist.Id = id
	}

	request := &v1.DeleteProcessWhitelistsRequest{
		Query: "",
	}
	_, err := suite.service.DeleteProcessWhitelists(hasWriteCtx, request)
	suite.Error(err)

	request = &v1.DeleteProcessWhitelistsRequest{
		Query:   "Deployment Id:d1",
		Confirm: false,
	}
	resp, err := suite.service.DeleteProcessWhitelists(hasWriteCtx, request)
	suite.NoError(err)
	suite.Equal(&v1.DeleteProcessWhitelistsResponse{
		NumDeleted: 1,
		DryRun:     true,
	}, resp)

	// Delete d1
	request.Confirm = true
	resp, err = suite.service.DeleteProcessWhitelists(hasWriteCtx, request)
	suite.NoError(err)
	suite.Equal(&v1.DeleteProcessWhitelistsResponse{
		NumDeleted: 1,
		DryRun:     false,
	}, resp)

	// Ensure that a second request doesn't return any values deleted
	resp, err = suite.service.DeleteProcessWhitelists(hasWriteCtx, request)
	suite.NoError(err)
	suite.Equal(&v1.DeleteProcessWhitelistsResponse{
		NumDeleted: 0,
		DryRun:     false,
	}, resp)

	// Delete d2 with a generic wildcard on deployment id
	request = &v1.DeleteProcessWhitelistsRequest{
		Query:   "Deployment Id:*",
		Confirm: true,
	}
	resp, err = suite.service.DeleteProcessWhitelists(hasWriteCtx, request)
	suite.NoError(err)
	suite.Equal(&v1.DeleteProcessWhitelistsResponse{
		NumDeleted: 1,
		DryRun:     false,
	}, resp)
}
