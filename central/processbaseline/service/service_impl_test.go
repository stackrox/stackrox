//go:build sql_integration

package service

import (
	"context"
	"testing"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	"github.com/stackrox/rox/central/processbaseline/datastore"
	baselineSearch "github.com/stackrox/rox/central/processbaseline/search"
	postgresStore "github.com/stackrox/rox/central/processbaseline/store/postgres"
	resultsMocks "github.com/stackrox/rox/central/processbaselineresults/datastore/mocks"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/central/reprocessor/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
	hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
)

func fillDB(t *testing.T, ds datastore.DataStore, baselines []*storage.ProcessBaseline) {
	for _, baseline := range baselines {
		_, err := ds.AddProcessBaseline(hasWriteCtx, baseline)
		assert.NoError(t, err)
	}
}

func emptyDB(t *testing.T, ds datastore.DataStore, baselines []*storage.ProcessBaseline) {
	for _, baseline := range baselines {
		assert.NoError(t, ds.RemoveProcessBaseline(hasWriteCtx, baseline.GetKey()))
	}
}

func getIndicators(key *storage.ProcessBaselineKey) []*storage.ProcessIndicator {
	return []*storage.ProcessIndicator{
		{
			Signal: &storage.ProcessSignal{
				ExecFilePath: "/bin/not-apt-get",
				Args:         "install nmap",
			},
			ContainerName: key.GetContainerName(),
		},
		{
			Signal: &storage.ProcessSignal{
				ExecFilePath: "/bin/apt-get",
				Args:         "install nmap",
			},
			ContainerName: key.GetContainerName(),
		},
		{
			Signal: &storage.ProcessSignal{
				ExecFilePath: "/bin/curl",
				Args:         "badssl.com",
			},
			ContainerName: key.GetContainerName(),
		},
	}
}

func TestProcessBaselineService(t *testing.T) {
	suite.Run(t, new(ProcessBaselineServiceTestSuite))
}

type ProcessBaselineServiceTestSuite struct {
	suite.Suite
	datastore datastore.DataStore
	service   Service

	pool postgres.DB

	reprocessor        *mocks.MockLoop
	resultDatastore    *resultsMocks.MockDataStore
	indicatorMockStore *indicatorMocks.MockDataStore
	connectionMgr      *connectionMocks.MockManager
	mockCtrl           *gomock.Controller
	deployments        *deploymentMocks.MockDataStore
	lifecycleManager   *lifecycleMocks.MockManager
}

func (suite *ProcessBaselineServiceTestSuite) SetupTest() {
	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB
	dbStore := postgresStore.New(suite.pool)
	cache, err := postgresStore.NewWithCache(dbStore)
	suite.NoError(err)
	store := cache
	indexer := postgresStore.NewIndexer(suite.pool)

	searcher, err := baselineSearch.New(store, indexer)
	suite.NoError(err)

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.resultDatastore = resultsMocks.NewMockDataStore(suite.mockCtrl)
	suite.resultDatastore.EXPECT().DeleteBaselineResults(gomock.Any(), gomock.Any()).AnyTimes()

	suite.indicatorMockStore = indicatorMocks.NewMockDataStore(suite.mockCtrl)
	suite.datastore = datastore.New(store, searcher, suite.resultDatastore, suite.indicatorMockStore)
	suite.reprocessor = mocks.NewMockLoop(suite.mockCtrl)
	suite.connectionMgr = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.deployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.lifecycleManager = lifecycleMocks.NewMockManager(suite.mockCtrl)
	suite.service = New(suite.datastore, suite.reprocessor, suite.connectionMgr, suite.deployments, suite.lifecycleManager)
}

func (suite *ProcessBaselineServiceTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
	suite.pool.Close()
}

func (suite *ProcessBaselineServiceTestSuite) TestGetProcessBaseline() {
	knownBaseline := fixtures.GetProcessBaselineWithKey()
	cases := []struct {
		name           string
		baselines      []*storage.ProcessBaseline
		expectedResult *storage.ProcessBaseline
		shouldFail     bool
	}{
		{
			name:       "Empty db",
			baselines:  []*storage.ProcessBaseline{},
			shouldFail: true,
		},
		{
			name:           "One process baseline",
			baselines:      []*storage.ProcessBaseline{knownBaseline},
			expectedResult: knownBaseline,
			shouldFail:     false,
		},
		{
			name: "Many process baselines",
			baselines: []*storage.ProcessBaseline{
				knownBaseline,
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
			},
			expectedResult: knownBaseline,
			shouldFail:     false,
		},
		{
			name: "Search for non-existent",
			baselines: []*storage.ProcessBaseline{
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
				fixtures.GetProcessBaselineWithKey(),
			},
			shouldFail: true,
		},
	}
	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			fillDB(t, suite.datastore, c.baselines)
			defer emptyDB(t, suite.datastore, c.baselines)
			requestByKey := &v1.GetProcessBaselineRequest{Key: knownBaseline.GetKey()}
			suite.deployments.EXPECT().GetDeployment(hasReadCtx, gomock.Any()).Return(nil, true, nil).AnyTimes()
			baseline, err := suite.service.GetProcessBaseline(hasReadCtx, requestByKey)
			if c.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedResult, baseline)
			}
		})
	}
}

func (suite *ProcessBaselineServiceTestSuite) TestGetLoadProcessBaseline() {
	knownBaseline := fixtures.GetProcessBaselineWithKey()
	key := knownBaseline.GetKey()
	indicators := getIndicators(key)

	requestByKey := &v1.GetProcessBaselineRequest{Key: knownBaseline.GetKey()}
	suite.deployments.EXPECT().GetDeployment(hasWriteCtx, gomock.Any()).Return(nil, true, nil)
	suite.indicatorMockStore.EXPECT().SearchRawProcessIndicators(hasWriteCtx, gomock.Any()).Return(indicators, nil)

	baseline, _ := suite.service.GetProcessBaseline(hasWriteCtx, requestByKey)

	assert.Equal(suite.T(), baseline.GetKey(), knownBaseline.GetKey())
}

func (suite *ProcessBaselineServiceTestSuite) TestGetLoadProcessBaselineDeletedDeployment() {
	knownBaseline := fixtures.GetProcessBaselineWithKey()

	requestByKey := &v1.GetProcessBaselineRequest{Key: knownBaseline.GetKey()}
	suite.deployments.EXPECT().GetDeployment(hasWriteCtx, gomock.Any()).Return(nil, false, nil)

	baseline, _ := suite.service.GetProcessBaseline(hasWriteCtx, requestByKey)

	assert.Nil(suite.T(), baseline)
}

func (suite *ProcessBaselineServiceTestSuite) TestUpdateProcessBaseline() {
	stockProcesses := []string{"stock_process_1", "stock_process_2"}

	baselineCollection := make(map[int]*storage.ProcessBaseline)
	getBaseline := func(index int) *storage.ProcessBaseline {
		if baseline, ok := baselineCollection[index]; ok {
			return baseline
		}
		baseline := fixtures.GetProcessBaselineWithKey()
		baseline.Elements = make([]*storage.BaselineElement, 0, len(stockProcesses))
		for _, stockProcess := range stockProcesses {
			baseline.Elements = append(baseline.Elements, &storage.BaselineElement{
				Element: &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: stockProcess}},
			})
		}
		baselineCollection[index] = baseline
		return baseline
	}

	getBaselines := func(indexes ...int) []*storage.ProcessBaseline {
		baselines := make([]*storage.ProcessBaseline, 0, len(indexes))
		for _, i := range indexes {
			baselines = append(baselines, getBaseline(i))
		}
		return baselines
	}

	getBaselineKey := func(index int) *storage.ProcessBaselineKey {
		return getBaseline(index).GetKey()
	}

	getBaselineKeys := func(indexes ...int) []*storage.ProcessBaselineKey {
		keys := make([]*storage.ProcessBaselineKey, 0, len(indexes))
		for _, i := range indexes {
			keys = append(keys, getBaselineKey(i))
		}
		return keys
	}

	cases := []struct {
		name                string
		baselines           []*storage.ProcessBaseline
		toUpdate            []*storage.ProcessBaselineKey
		toAdd               []string
		toRemove            []string
		expectedSuccessKeys []*storage.ProcessBaselineKey
		expectedErrorKeys   []*storage.ProcessBaselineKey
	}{
		{
			name:              "Update non-existent",
			toUpdate:          getBaselineKeys(0, 1),
			toAdd:             []string{"Doesn't matter"},
			toRemove:          []string{"whatever"},
			expectedErrorKeys: getBaselineKeys(0, 1),
		},
		{
			name:                "Update one",
			baselines:           getBaselines(0),
			toUpdate:            getBaselineKeys(0),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[0]},
			expectedSuccessKeys: getBaselineKeys(0),
		},
		{
			name:                "Update many",
			baselines:           getBaselines(0, 1, 2, 3, 4),
			toUpdate:            getBaselineKeys(0, 1, 2, 3, 4),
			toAdd:               []string{"Some process"},
			expectedSuccessKeys: getBaselineKeys(0, 1, 2, 3, 4),
		},
		{
			name:                "Mixed failures",
			baselines:           getBaselines(0),
			toUpdate:            getBaselineKeys(0, 1),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[0]},
			expectedSuccessKeys: getBaselineKeys(0),
			expectedErrorKeys:   getBaselineKeys(1),
		},
		{
			name:                "Unrelated list",
			baselines:           getBaselines(0, 1),
			toUpdate:            getBaselineKeys(0),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[1]},
			expectedSuccessKeys: getBaselineKeys(0),
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			fillDB(t, suite.datastore, c.baselines)
			defer emptyDB(t, suite.datastore, c.baselines)

			request := &v1.UpdateProcessBaselinesRequest{
				Keys:           c.toUpdate,
				AddElements:    fixtures.MakeBaselineItems(c.toAdd...),
				RemoveElements: fixtures.MakeBaselineItems(c.toRemove...),
			}
			suite.reprocessor.EXPECT().ReprocessRiskForDeployments(gomock.Any())
			for range c.expectedSuccessKeys {
				suite.connectionMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any())
			}
			response, err := suite.service.UpdateProcessBaselines(hasWriteCtx, request)
			assert.NoError(t, err)
			var successKeys []*storage.ProcessBaselineKey
			for _, wl := range response.Baselines {
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
					if sliceutils.Find(c.toRemove, stockProcess) == -1 {
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

func (suite *ProcessBaselineServiceTestSuite) TestDeleteProcessBaselines() {
	baselines := []*storage.ProcessBaseline{
		{
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  fixtureconsts.Deployment1,
				ContainerName: "container",
				ClusterId:     fixtureconsts.Cluster1,
				Namespace:     "namespace",
			},
			Elements: []*storage.BaselineElement{
				{
					Element: &storage.BaselineItem{
						Item: &storage.BaselineItem_ProcessName{
							ProcessName: "d1_process",
						},
					},
				},
			},
		},
		{
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  fixtureconsts.Deployment2,
				ContainerName: "container",
				ClusterId:     fixtureconsts.Cluster1,
				Namespace:     "namespace",
			},
			Elements: []*storage.BaselineElement{
				{
					Element: &storage.BaselineItem{
						Item: &storage.BaselineItem_ProcessName{
							ProcessName: "d2_process",
						},
					},
				},
			},
		},
	}

	suite.deployments.EXPECT().GetDeployment(hasWriteCtx, gomock.Any()).Return(nil, true, nil).AnyTimes()
	suite.lifecycleManager.EXPECT().RemoveDeploymentFromObservation(fixtureconsts.Deployment1).AnyTimes()
	suite.lifecycleManager.EXPECT().RemoveDeploymentFromObservation(fixtureconsts.Deployment2).AnyTimes()

	for _, baseline := range baselines {
		id, err := suite.datastore.AddProcessBaseline(hasWriteCtx, baseline)
		suite.NoError(err)
		baseline.Id = id
	}

	request := &v1.DeleteProcessBaselinesRequest{
		Query: "",
	}
	_, err := suite.service.DeleteProcessBaselines(hasWriteCtx, request)
	suite.Error(err)

	request = &v1.DeleteProcessBaselinesRequest{
		Query:   "Deployment Id:" + fixtureconsts.Deployment1,
		Confirm: false,
	}
	resp, err := suite.service.DeleteProcessBaselines(hasWriteCtx, request)
	suite.NoError(err)
	suite.Equal(&v1.DeleteProcessBaselinesResponse{
		NumDeleted: 1,
		DryRun:     true,
	}, resp)
	requestByKey := &v1.GetProcessBaselineRequest{Key: baselines[0].Key}
	baseline, _ := suite.service.GetProcessBaseline(hasReadCtx, requestByKey)
	suite.NotNil(baseline.Elements)

	// Delete d1
	request.Confirm = true
	resp, err = suite.service.DeleteProcessBaselines(hasWriteCtx, request)
	suite.NoError(err)
	suite.Equal(&v1.DeleteProcessBaselinesResponse{
		NumDeleted: 1,
		DryRun:     false,
	}, resp)

	// Make sure the baseline exists, but it is empty i.e. no elements
	requestByKey = &v1.GetProcessBaselineRequest{Key: baselines[0].Key}
	baseline, _ = suite.service.GetProcessBaseline(hasReadCtx, requestByKey)
	suite.Empty(baseline.Elements)

	// Delete d2 with a generic wildcard on deployment id
	request = &v1.DeleteProcessBaselinesRequest{
		Query:   "Deployment Id:*",
		Confirm: true,
	}
	resp, err = suite.service.DeleteProcessBaselines(hasWriteCtx, request)
	suite.NoError(err)
	suite.Equal(&v1.DeleteProcessBaselinesResponse{
		NumDeleted: int32(len(baselines)),
		DryRun:     false,
	}, resp)
}
