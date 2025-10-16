//go:build sql_integration

package evaluator

import (
	"context"
	"testing"
	"time"

	baselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	resultDatastore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	indicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processindicator/views"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var log = logging.LoggerForModule()

func TestProcessBaselineEvaluatorIntegration(t *testing.T) {
	suite.Run(t, new(ProcessBaselineEvaluatorIntegrationTestSuite))
}

type ProcessBaselineEvaluatorIntegrationTestSuite struct {
	suite.Suite

	pool postgres.DB

	baselinesDatastore  baselineDatastore.DataStore
	resultsDatastore    resultDatastore.DataStore
	indicatorsDatastore indicatorDatastore.DataStore

	evaluator Evaluator

	ctx context.Context
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) SetupSuite() {
	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB

	// Create real datastores
	suite.baselinesDatastore = baselineDatastore.GetTestPostgresDataStore(suite.T(), suite.pool)
	suite.resultsDatastore = resultDatastore.GetTestPostgresDataStore(suite.T(), suite.pool)
	suite.indicatorsDatastore = indicatorDatastore.GetTestPostgresDataStore(suite.T(), suite.pool)

	suite.evaluator = New(suite.resultsDatastore, suite.baselinesDatastore, suite.indicatorsDatastore)

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension),
		),
	)
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TearDownSuite() {
	if suite.indicatorsDatastore != nil {
		suite.indicatorsDatastore.Stop()
	}
	if suite.pool != nil {
		suite.pool.Close()
	}
}

// addLockedBaseline adds a baseline and locks it using UserLock
func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) addLockedBaseline(baseline *storage.ProcessBaseline) {
	_, err := suite.baselinesDatastore.AddProcessBaseline(suite.ctx, baseline)
	suite.NoError(err)

	// If we want it locked, use UserLockProcessBaseline which locks it immediately
	if baseline.GetUserLockedTimestamp() != nil {
		_, err = suite.baselinesDatastore.UserLockProcessBaseline(suite.ctx, baseline.GetKey(), true)
		suite.NoError(err)
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestNoProcessBaseline() {
	deployment := fixtures.GetDeployment()

	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Empty(results)

	// Verify the result was persisted
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Equal(deployment.GetId(), persistedResult.GetDeploymentId())
	suite.Equal(deployment.GetClusterId(), persistedResult.GetClusterId())
	suite.Equal(deployment.GetNamespace(), persistedResult.GetNamespace())
	suite.Len(persistedResult.GetBaselineStatuses(), 2)

	for _, status := range persistedResult.GetBaselineStatuses() {
		suite.Equal(storage.ContainerNameAndBaselineStatus_NOT_GENERATED, status.GetBaselineStatus())
		suite.False(status.GetAnomalousProcessesExecuted())
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestProcessBaselineExistsButNotLocked() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName := deployment.GetContainers()[0].GetName()

	// Create an unlocked baseline
	baseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		Elements: []*storage.BaselineElement{},
	}
	_, err := suite.baselinesDatastore.AddProcessBaseline(suite.ctx, baseline)
	suite.NoError(err)

	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Empty(results)

	// Verify the result was persisted
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Len(persistedResult.GetBaselineStatuses(), 2)

	for _, status := range persistedResult.GetBaselineStatuses() {
		if status.GetContainerName() == containerName {
			suite.Equal(storage.ContainerNameAndBaselineStatus_UNLOCKED, status.GetBaselineStatus())
		} else {
			suite.Equal(storage.ContainerNameAndBaselineStatus_NOT_GENERATED, status.GetBaselineStatus())
		}
		suite.False(status.GetAnomalousProcessesExecuted())
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestLockedProcessBaselineAllProcessesInBaseline() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName := deployment.GetContainers()[0].GetName()

	// Create a locked baseline with processes
	baseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            fixtures.MakeBaselineElements("/bin/apt-get", "/unrelated"),
	}
	suite.addLockedBaseline(baseline)

	// Add a process indicator that is in the baseline
	var err error
	processIndicator := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "/bin/apt-get",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}
	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator)
	suite.NoError(err)

	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Empty(results)

	// Verify the result was persisted
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Len(persistedResult.GetBaselineStatuses(), 2)
	log.Infof("SHREWS -- %v", persistedResult)

	for _, status := range persistedResult.GetBaselineStatuses() {
		// We only locked the first container
		if status.GetContainerName() == containerName {
			suite.Equal(storage.ContainerNameAndBaselineStatus_LOCKED, status.GetBaselineStatus())
			suite.False(status.GetAnomalousProcessesExecuted())
		} else {
			suite.Equal(storage.ContainerNameAndBaselineStatus_NOT_GENERATED, status.GetBaselineStatus())
			suite.False(status.GetAnomalousProcessesExecuted())
		}
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestLockedProcessBaselineOneNotInBaselineProcess() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName := deployment.GetContainers()[0].GetName()

	// Create a locked baseline without the process we'll add
	baseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            []*storage.BaselineElement{},
	}
	suite.addLockedBaseline(baseline)

	// Add a process indicator that is NOT in the baseline
	var err error
	processIndicator := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "apt-get",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}
	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator)
	suite.NoError(err)

	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("apt-get", results[0].ExecFilePath)
	suite.Equal("install nmap", results[0].SignalArgs)
	suite.Equal(containerName, results[0].ContainerName)

	// Verify the result was persisted
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Len(persistedResult.GetBaselineStatuses(), 2)

	// Check first container has anomalous process, second doesn't
	for _, status := range persistedResult.GetBaselineStatuses() {
		if status.GetContainerName() == containerName {
			suite.True(status.GetAnomalousProcessesExecuted())
			suite.Equal(storage.ContainerNameAndBaselineStatus_LOCKED, status.GetBaselineStatus())
		} else {
			suite.False(status.GetAnomalousProcessesExecuted())
			suite.Equal(storage.ContainerNameAndBaselineStatus_NOT_GENERATED, status.GetBaselineStatus())
		}
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestLockedProcessBaselineTwoNotInBaselineProcesses() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName := deployment.GetContainers()[1].GetName()

	// Create a locked baseline without the processes we'll add
	baseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            []*storage.BaselineElement{},
	}
	suite.addLockedBaseline(baseline)

	// Add two process indicators that are NOT in the baseline
	var err error
	processIndicator1 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "apt-get",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	processIndicator2 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "curl",
			Args:         "badssl.com",
			ExecFilePath: "curl",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator1, processIndicator2)
	suite.NoError(err)

	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Len(results, 2)

	// Verify the result was persisted
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Len(persistedResult.GetBaselineStatuses(), 2)

	// Check second container has anomalous processes, first doesn't
	for _, status := range persistedResult.GetBaselineStatuses() {
		if status.GetContainerName() == containerName {
			suite.True(status.GetAnomalousProcessesExecuted())
			suite.Equal(storage.ContainerNameAndBaselineStatus_LOCKED, status.GetBaselineStatus())
		} else {
			suite.False(status.GetAnomalousProcessesExecuted())
			suite.Equal(storage.ContainerNameAndBaselineStatus_NOT_GENERATED, status.GetBaselineStatus())
		}
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestLockedProcessBaselineTwoContainersDifferentProcesses() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName1 := deployment.GetContainers()[0].GetName()
	containerName2 := deployment.GetContainers()[1].GetName()

	// Create locked baselines for both containers
	baseline1 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName1,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            fixtures.MakeBaselineElements("/bin/apt-get"),
	}
	suite.addLockedBaseline(baseline1)

	baseline2 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName2,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            []*storage.BaselineElement{},
	}
	suite.addLockedBaseline(baseline2)

	// Add process indicators: one not in baseline for container 0, one in baseline, one not in baseline for container 1
	var err error
	processIndicator1 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName1,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "not-apt-get",
			Args:         "install nmap",
			ExecFilePath: "/bin/not-apt-get",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	processIndicator2 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName1,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "/bin/apt-get",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	processIndicator3 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName2,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "curl",
			Args:         "badssl.com",
			ExecFilePath: "/bin/curl",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator1, processIndicator2, processIndicator3)
	suite.NoError(err)

	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Len(results, 2)

	// Check that we got the right violations
	paths := []string{results[0].ExecFilePath, results[1].ExecFilePath}
	suite.Contains(paths, "/bin/not-apt-get")
	suite.Contains(paths, "/bin/curl")

	// Verify the result was persisted
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Len(persistedResult.GetBaselineStatuses(), 2)

	// Both containers have anomalous processes
	for _, status := range persistedResult.GetBaselineStatuses() {
		suite.Equal(storage.ContainerNameAndBaselineStatus_LOCKED, status.GetBaselineStatus())
		suite.True(status.GetAnomalousProcessesExecuted())
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestResultAlreadyExistsNoUpdate() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName1 := deployment.GetContainers()[0].GetName()
	containerName2 := deployment.GetContainers()[1].GetName()

	// Create locked baselines
	baseline1 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName1,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            fixtures.MakeBaselineElements("/bin/apt-get"),
	}
	suite.addLockedBaseline(baseline1)

	baseline2 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName2,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            []*storage.BaselineElement{},
	}
	suite.addLockedBaseline(baseline2)

	// Add process indicators
	var err error
	processIndicator1 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName1,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "not-apt-get",
			Args:         "install nmap",
			ExecFilePath: "/bin/not-apt-get",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	processIndicator2 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName2,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "curl",
			Args:         "badssl.com",
			ExecFilePath: "/bin/curl",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator1, processIndicator2)
	suite.NoError(err)

	// Create the existing result that matches what we expect
	existingResult := &storage.ProcessBaselineResults{
		DeploymentId: deployment.GetId(),
		ClusterId:    deployment.GetClusterId(),
		Namespace:    deployment.GetNamespace(),
		BaselineStatuses: []*storage.ContainerNameAndBaselineStatus{
			{
				ContainerName:              containerName2,
				BaselineStatus:             storage.ContainerNameAndBaselineStatus_LOCKED,
				AnomalousProcessesExecuted: true,
			},
			{
				ContainerName:              containerName1,
				BaselineStatus:             storage.ContainerNameAndBaselineStatus_LOCKED,
				AnomalousProcessesExecuted: true,
			},
		},
	}
	err = suite.resultsDatastore.UpsertBaselineResults(suite.ctx, existingResult)
	suite.NoError(err)

	// Get initial timestamp
	initialResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(initialResult)

	// Run the evaluator
	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Len(results, 2)

	// The result should still be the same (no update needed)
	// Note: we can't easily test that UpsertBaselineResults was NOT called since we're using real datastores
	// but we can verify the result is still correct
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Len(persistedResult.GetBaselineStatuses(), 2)
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestResultAlreadyExistsNeedsUpdate() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName1 := deployment.GetContainers()[0].GetName()
	containerName2 := deployment.GetContainers()[1].GetName()

	// Create locked baselines
	baseline1 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName1,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            fixtures.MakeBaselineElements("/bin/apt-get"),
	}
	suite.addLockedBaseline(baseline1)

	baseline2 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName2,
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            []*storage.BaselineElement{},
	}
	suite.addLockedBaseline(baseline2)

	// Add process indicators
	var err error
	processIndicator1 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName1,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "not-apt-get",
			Args:         "install nmap",
			ExecFilePath: "/bin/not-apt-get",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	processIndicator2 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName2,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "curl",
			Args:         "badssl.com",
			ExecFilePath: "/bin/curl",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}

	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator1, processIndicator2)
	suite.NoError(err)

	// Create an existing result that needs update (UNLOCKED instead of LOCKED for container 1)
	existingResult := &storage.ProcessBaselineResults{
		DeploymentId: deployment.GetId(),
		ClusterId:    deployment.GetClusterId(),
		Namespace:    deployment.GetNamespace(),
		BaselineStatuses: []*storage.ContainerNameAndBaselineStatus{
			{
				ContainerName:              containerName1,
				BaselineStatus:             storage.ContainerNameAndBaselineStatus_UNLOCKED, // Different from expected
				AnomalousProcessesExecuted: true,
			},
			{
				ContainerName:              containerName2,
				BaselineStatus:             storage.ContainerNameAndBaselineStatus_LOCKED,
				AnomalousProcessesExecuted: true,
			},
		},
	}
	err = suite.resultsDatastore.UpsertBaselineResults(suite.ctx, existingResult)
	suite.NoError(err)

	// Run the evaluator
	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Len(results, 2)

	// Verify the result was updated
	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.NotNil(persistedResult)
	suite.Len(persistedResult.GetBaselineStatuses(), 2)

	// Check that all statuses are now LOCKED
	for _, status := range persistedResult.GetBaselineStatuses() {
		suite.Equal(storage.ContainerNameAndBaselineStatus_LOCKED, status.GetBaselineStatus())
		suite.True(status.GetAnomalousProcessesExecuted())
	}
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestComplexWorkflow() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName := deployment.GetContainers()[0].GetName()

	// Start with no baseline - should get NOT_GENERATED
	results, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Empty(results)

	persistedResult, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.Equal(storage.ContainerNameAndBaselineStatus_NOT_GENERATED, persistedResult.GetBaselineStatuses()[0].GetBaselineStatus())

	// Create a locked baseline
	baseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: deployment.GetContainers()[0].GetName(),
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            fixtures.MakeBaselineElements("/bin/allowed"),
	}
	suite.addLockedBaseline(baseline)

	// Add an anomalous process
	processIndicator := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment.GetId(),
		ContainerName: containerName,
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "malicious",
			Args:         "badargs",
			ExecFilePath: "/bin/malicious",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}
	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator)
	suite.NoError(err)

	// Should now detect the anomalous process
	results, err = suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("/bin/malicious", results[0].ExecFilePath)

	persistedResult, err = suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.Equal(storage.ContainerNameAndBaselineStatus_LOCKED, persistedResult.GetBaselineStatuses()[0].GetBaselineStatus())
	suite.True(persistedResult.GetBaselineStatuses()[0].GetAnomalousProcessesExecuted())

	// Remove the anomalous process
	err = suite.indicatorsDatastore.RemoveProcessIndicators(suite.ctx, []string{processIndicator.GetId()})
	suite.NoError(err)

	// Should now show no anomalous processes
	results, err = suite.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	suite.NoError(err)
	suite.Empty(results)

	persistedResult, err = suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment.GetId())
	suite.NoError(err)
	suite.Equal(storage.ContainerNameAndBaselineStatus_LOCKED, persistedResult.GetBaselineStatuses()[0].GetBaselineStatus())
	suite.False(persistedResult.GetBaselineStatuses()[0].GetAnomalousProcessesExecuted())
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestQueryProcessIndicators() {
	deployment := fixtures.GetDeployment()
	deployment.Id = uuid.NewV4().String()
	containerName1 := deployment.GetContainers()[0].GetName()
	containerName2 := deployment.GetContainers()[1].GetName()

	// Add multiple process indicators for the deployment
	indicators := []*storage.ProcessIndicator{
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName1,
			PodId:         uuid.NewV4().String(),
			PodUid:        uuid.NewV4().String(),
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
			Signal: &storage.ProcessSignal{
				Name:         "process1",
				Args:         "args1",
				ExecFilePath: "/bin/process1",
				Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
				ContainerId:  uuid.NewV4().String(),
				Uid:          1000,
			},
			ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
		},
		{
			Id:            uuid.NewV4().String(),
			DeploymentId:  deployment.GetId(),
			ContainerName: containerName2,
			PodId:         uuid.NewV4().String(),
			PodUid:        uuid.NewV4().String(),
			ClusterId:     deployment.GetClusterId(),
			Namespace:     deployment.GetNamespace(),
			Signal: &storage.ProcessSignal{
				Name:         "process2",
				Args:         "args2",
				ExecFilePath: "/bin/process2",
				Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
				ContainerId:  uuid.NewV4().String(),
				Uid:          1000,
			},
			ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
		},
	}

	err := suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, indicators...)
	suite.NoError(err)

	// Query for indicators by deployment ID
	query := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deployment.GetId()).ProtoQuery()
	riskViews := make([]*views.ProcessIndicatorRiskView, 0, len(indicators))
	err = suite.indicatorsDatastore.IterateOverProcessIndicatorsRiskView(suite.ctx, query, func(view *views.ProcessIndicatorRiskView) error {
		riskViews = append(riskViews, view)
		return nil
	})
	suite.NoError(err)
	suite.Len(riskViews, 2)

	// Verify the risk views have the expected data
	paths := []string{riskViews[0].ExecFilePath, riskViews[1].ExecFilePath}
	suite.Contains(paths, "/bin/process1")
	suite.Contains(paths, "/bin/process2")
}

func (suite *ProcessBaselineEvaluatorIntegrationTestSuite) TestMultipleDeploymentsIsolation() {
	deployment1 := fixtures.GetDeployment()
	deployment1.Id = uuid.NewV4().String()
	deployment2 := fixtures.GetDeployment()
	deployment2.Id = uuid.NewV4().String()

	// Create baselines for both deployments
	baseline1 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment1.GetId(),
			ContainerName: deployment1.GetContainers()[0].GetName(),
			ClusterId:     deployment1.GetClusterId(),
			Namespace:     deployment1.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            []*storage.BaselineElement{},
	}
	suite.addLockedBaseline(baseline1)

	baseline2 := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deployment2.GetId(),
			ContainerName: deployment2.GetContainers()[0].GetName(),
			ClusterId:     deployment2.GetClusterId(),
			Namespace:     deployment2.GetNamespace(),
		},
		UserLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
		Elements:            []*storage.BaselineElement{},
	}
	suite.addLockedBaseline(baseline2)

	// Add process indicators for deployment1 only
	var err error
	processIndicator1 := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  deployment1.GetId(),
		ContainerName: deployment1.GetContainers()[0].GetName(),
		PodId:         uuid.NewV4().String(),
		PodUid:        uuid.NewV4().String(),
		ClusterId:     deployment1.GetClusterId(),
		Namespace:     deployment1.GetNamespace(),
		Signal: &storage.ProcessSignal{
			Name:         "malicious",
			Args:         "badargs",
			ExecFilePath: "/bin/malicious",
			Time:         protoconv.ConvertTimeToTimestamp(time.Now()),
			ContainerId:  uuid.NewV4().String(),
			Uid:          1000,
		},
		ContainerStartTime: protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * time.Hour)),
	}
	err = suite.indicatorsDatastore.AddProcessIndicators(suite.ctx, processIndicator1)
	suite.NoError(err)

	// Evaluate deployment1 - should have violations
	results1, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment1)
	suite.NoError(err)
	suite.Len(results1, 1)

	// Evaluate deployment2 - should have no violations
	results2, err := suite.evaluator.EvaluateBaselinesAndPersistResult(deployment2)
	suite.NoError(err)
	suite.Empty(results2)

	// Verify results are isolated
	persistedResult1, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment1.GetId())
	suite.NoError(err)
	suite.True(persistedResult1.GetBaselineStatuses()[0].GetAnomalousProcessesExecuted())

	persistedResult2, err := suite.resultsDatastore.GetBaselineResults(suite.ctx, deployment2.GetId())
	suite.NoError(err)
	suite.False(persistedResult2.GetBaselineStatuses()[0].GetAnomalousProcessesExecuted())
}
