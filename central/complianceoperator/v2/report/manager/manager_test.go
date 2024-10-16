package manager

import (
	"context"
	"testing"
	"time"

	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/complianceReportgenerator/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ManagerTestSuite struct {
	suite.Suite
	mockCtrl            *gomock.Controller
	ctx                 context.Context
	scanConfigDataStore *scanConfigurationDS.MockDataStore
	scanDataStore       *scanMocks.MockDataStore
	profileDataStore    *profileMocks.MockDataStore
	snapshotDataStore   *snapshotMocks.MockDataStore
	reportGen           *reportGen.MockComplianceReportGenerator
}

func (m *ManagerTestSuite) SetupSuite() {
	m.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	m.ctx = sac.WithAllAccess(context.Background())
}

func (m *ManagerTestSuite) SetupTest() {
	m.mockCtrl = gomock.NewController(m.T())
	m.scanConfigDataStore = scanConfigurationDS.NewMockDataStore(m.mockCtrl)
	m.scanDataStore = scanMocks.NewMockDataStore(m.mockCtrl)
	m.profileDataStore = profileMocks.NewMockDataStore(m.mockCtrl)
	m.snapshotDataStore = snapshotMocks.NewMockDataStore(m.mockCtrl)
	m.reportGen = reportGen.NewMockComplianceReportGenerator(m.mockCtrl)
}

func TestComplianceReportManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (m *ManagerTestSuite) TestSubmitReportRequest() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.reportGen)
	reportRequest := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "test_scan_config",
		Id:             "test_scan_config",
	}
	err := manager.SubmitReportRequest(m.ctx, reportRequest)
	m.Require().NoError(err)
	err = manager.SubmitReportRequest(m.ctx, reportRequest)
	m.Require().Error(err)
}

func (m *ManagerTestSuite) TearDownTest() {
	m.mockCtrl.Finish()
}

func (m *ManagerTestSuite) TestHandleScan() {
	m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
			return []*storage.ComplianceOperatorReportSnapshotV2{}, nil
		})
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.reportGen)
	managerImplementation, ok := manager.(*managerImpl)
	require.True(m.T(), ok)
	scan := &storage.ComplianceOperatorScanV2{
		ClusterId: "cluster-id",
	}
	err := manager.HandleScan(context.Background(), scan)
	assert.Error(m.T(), err)

	scan.Id = "scan-id"
	err = manager.HandleScan(context.Background(), scan)
	assert.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		assert.Len(m.T(), managerImplementation.watchingScans, 0)
	})

	scan.LastStartedTime = protocompat.TimestampNow()
	err = manager.HandleScan(context.Background(), scan)
	assert.NoError(m.T(), err)
	id, err := watcher.GetWatcherIDFromScan(context.Background(), scan, m.snapshotDataStore, nil)
	require.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
	})
}

func (m *ManagerTestSuite) TestHandleResult() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.reportGen)
	managerImplementation, ok := manager.(*managerImpl)
	require.True(m.T(), ok)
	timeNow := time.Now()
	pastTime := timeNow.Add(-10 * time.Second)
	futureTime := timeNow.Add(10 * time.Second)
	timeNowProto, err := protocompat.ConvertTimeToTimestampOrError(timeNow)
	require.NoError(m.T(), err)
	nowRFCFormat := timeNow.Format(time.RFC3339Nano)
	pastRFCFormat := pastTime.Format(time.RFC3339Nano)
	futureRFCFormat := futureTime.Format(time.RFC3339Nano)
	result := &storage.ComplianceOperatorCheckResultV2{
		Annotations: map[string]string{
			"compliance.openshift.io/last-scanned-timestamp": pastRFCFormat,
		},
	}
	scan := &storage.ComplianceOperatorScanV2{
		ClusterId: "cluster-id",
		Id:        "scan-id",
	}
	m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(2).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
			return []*storage.ComplianceOperatorScanV2{scan}, nil
		})
	m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
			return []*storage.ComplianceOperatorReportSnapshotV2{}, nil
		})
	id, err := watcher.GetWatcherIDFromCheckResult(context.Background(), result, m.scanDataStore, m.snapshotDataStore)
	require.NoError(m.T(), err)
	err = manager.HandleResult(context.Background(), result)
	assert.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
		delete(managerImplementation.watchingScans, id)
	})

	scan.LastStartedTime = timeNowProto
	m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
			return []*storage.ComplianceOperatorScanV2{scan}, nil
		})

	err = manager.HandleResult(context.Background(), result)
	assert.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		assert.Len(m.T(), managerImplementation.watchingScans, 0)
	})

	result.Annotations["compliance.openshift.io/last-scanned-timestamp"] = nowRFCFormat
	err = manager.HandleResult(context.Background(), result)
	assert.NoError(m.T(), err)
	id, err = watcher.GetWatcherIDFromCheckResult(context.Background(), result, m.scanDataStore, m.snapshotDataStore)
	require.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
		delete(managerImplementation.watchingScans, id)
	})
	result.Annotations["compliance.openshift.io/last-scanned-timestamp"] = futureRFCFormat
	err = manager.HandleResult(context.Background(), result)
	assert.NoError(m.T(), err)
	id, err = watcher.GetWatcherIDFromCheckResult(context.Background(), result, m.scanDataStore, m.snapshotDataStore)
	require.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
		delete(managerImplementation.watchingScans, id)
	})
}
