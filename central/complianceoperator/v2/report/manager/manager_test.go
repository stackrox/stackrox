package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	checkResultsMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	integrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore/mocks"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ManagerTestSuite struct {
	suite.Suite
	mockCtrl                       *gomock.Controller
	ctx                            context.Context
	scanConfigDataStore            *scanConfigurationDS.MockDataStore
	scanDataStore                  *scanMocks.MockDataStore
	profileDataStore               *profileMocks.MockDataStore
	snapshotDataStore              *snapshotMocks.MockDataStore
	complianceIntegrationDataStore *integrationMocks.MockDataStore
	suiteDataStore                 *suiteDS.MockDataStore
	bindingsDataStore              *bindingsDS.MockDataStore
	checkResultDataStore           *checkResultsMocks.MockDataStore
	reportGen                      *reportGen.MockComplianceReportGenerator
}

func (m *ManagerTestSuite) SetupSuite() {
	m.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	m.T().Setenv(features.ScanScheduleReportJobs.EnvVar(), "true")
	m.ctx = sac.WithAllAccess(context.Background())
}

func (m *ManagerTestSuite) SetupTest() {
	m.mockCtrl = gomock.NewController(m.T())
	m.scanConfigDataStore = scanConfigurationDS.NewMockDataStore(m.mockCtrl)
	m.scanDataStore = scanMocks.NewMockDataStore(m.mockCtrl)
	m.profileDataStore = profileMocks.NewMockDataStore(m.mockCtrl)
	m.snapshotDataStore = snapshotMocks.NewMockDataStore(m.mockCtrl)
	m.complianceIntegrationDataStore = integrationMocks.NewMockDataStore(m.mockCtrl)
	m.suiteDataStore = suiteDS.NewMockDataStore(m.mockCtrl)
	m.bindingsDataStore = bindingsDS.NewMockDataStore(m.mockCtrl)
	m.checkResultDataStore = checkResultsMocks.NewMockDataStore(m.mockCtrl)
	m.reportGen = reportGen.NewMockComplianceReportGenerator(m.mockCtrl)
}

func TestComplianceReportManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (m *ManagerTestSuite) TestSubmitReportRequest() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
	reportRequest := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "test_scan_config",
		Id:             "test_scan_config",
	}
	err := manager.SubmitReportRequest(m.ctx, reportRequest, storage.ComplianceOperatorReportStatus_EMAIL)
	m.Require().NoError(err)
	err = manager.SubmitReportRequest(m.ctx, reportRequest, storage.ComplianceOperatorReportStatus_EMAIL)
	m.Require().Error(err)
}

func (m *ManagerTestSuite) TearDownTest() {
	m.mockCtrl.Finish()
}

func (m *ManagerTestSuite) TestHandleReportRequest() {
	m.T().Setenv(env.ReportExecutionMaxConcurrency.EnvVar(), "1")

	scanConfig := getTestScanConfig()

	newIdentityCtx := func() context.Context {
		identity := mocks.NewMockIdentity(m.mockCtrl)
		identity.EXPECT().UID().AnyTimes().Return("user-id")
		identity.EXPECT().FullName().AnyTimes().Return("user-name")
		identity.EXPECT().FriendlyName().AnyTimes().Return("user-friendly-name")
		return authn.ContextWithIdentity(context.Background(), identity, m.T())
	}

	setupReportDataMocks := func() {
		m.scanConfigDataStore.EXPECT().GetScanConfigClusterStatus(gomock.Any(), gomock.Eq(scanConfig.GetId())).
			Times(1).Return(getTestClusterStatusFromScanConfig(scanConfig), nil)
		m.suiteDataStore.EXPECT().GetSuites(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
		m.bindingsDataStore.EXPECT().GetScanSettingBindings(gomock.Any(), gomock.Any()).
			Times(len(scanConfig.GetClusters())).Return(nil, nil)
	}

	m.Run("No watcher running", func() {
		setupReportDataMocks()
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		m.snapshotDataStore.EXPECT().GetLastSnapshotFromScanConfig(gomock.Any(), gomock.Eq(scanConfig.GetId())).
			Times(1).Return(nil, nil)
		m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).
			Times(len(scanConfig.GetClusters())).
			Return(getScans(len(scanConfig.GetProfiles())), nil)
		m.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).Times(1).Return(nil)

		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		req := &reportRequest{
			scanConfig:         scanConfig,
			ctx:                newIdentityCtx(),
			notificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
		}
		generated, err := impl.handleReportRequest(req)
		m.Require().NoError(err)
		m.Assert().True(generated)
		m.Assert().NotEmpty(req.snapshotID)
	})

	m.Run("Upsert snapshot error", func() {
		setupReportDataMocks()
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(errors.New("db error"))

		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		req := &reportRequest{
			scanConfig:         scanConfig,
			ctx:                newIdentityCtx(),
			notificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
		}
		generated, err := impl.handleReportRequest(req)
		m.Require().Error(err)
		m.Assert().False(generated)
		m.Assert().Contains(err.Error(), "unable to upsert snapshot")
	})

	m.Run("Watcher running, subscribes and defers report", func() {
		setupReportDataMocks()
		stubWatcher := &stubScanConfigWatcher{}
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).Return(nil)

		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		concurrency.WithLock(&impl.watchingScanConfigsLock, func() {
			impl.watchingScanConfigs[scanConfig.GetId()] = stubWatcher
		})

		req := &reportRequest{
			scanConfig:         scanConfig,
			ctx:                newIdentityCtx(),
			notificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
		}
		generated, err := impl.handleReportRequest(req)
		m.Require().NoError(err)
		m.Assert().False(generated, "report should be deferred when watcher is running")
		m.Assert().Len(stubWatcher.subscribedSnapshots, 1)
		m.Assert().Equal(storage.ComplianceOperatorReportStatus_WAITING, stubWatcher.subscribedSnapshots[0].GetReportStatus().GetRunState())
	})

	m.Run("Missing identity returns error", func() {
		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		req := &reportRequest{
			scanConfig:         scanConfig,
			ctx:                context.Background(),
			notificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
		}
		generated, err := impl.handleReportRequest(req)
		m.Require().Error(err)
		m.Assert().False(generated)
		m.Assert().Contains(err.Error(), "could not determine user identity")
	})
}

func (m *ManagerTestSuite) TestGenerateReportFromWatcherResults() {
	m.T().Setenv(env.ReportExecutionMaxConcurrency.EnvVar(), "1")
	scanConfig := getTestScanConfig()

	setupReportDataMocks := func() {
		m.scanConfigDataStore.EXPECT().GetScanConfigClusterStatus(gomock.Any(), gomock.Eq(scanConfig.GetId())).
			Times(1).Return(getTestClusterStatusFromScanConfig(scanConfig), nil)
		m.suiteDataStore.EXPECT().GetSuites(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
		m.bindingsDataStore.EXPECT().GetScanSettingBindings(gomock.Any(), gomock.Any()).
			Times(len(scanConfig.GetClusters())).Return(nil, nil)
	}

	m.Run("Successful report generation", func() {
		setupReportDataMocks()
		m.complianceIntegrationDataStore.EXPECT().
			GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
		m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).
			Times(len(scanConfig.GetClusters())).
			Return(getScans(len(scanConfig.GetProfiles())), nil)
		done := make(chan struct{})
		m.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).Times(1).
			DoAndReturn(func(_ any) error {
				close(done)
				return nil
			})

		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		result := &watcher.ScanConfigWatcherResults{
			WatcherID:  scanConfig.GetId(),
			ScanConfig: scanConfig,
			ScanResults: map[string]*watcher.ScanWatcherResults{
				"cluster-1:scan-1": {Scan: &storage.ComplianceOperatorScanV2{Id: "scan-1", ClusterId: "cluster-1"}},
				"cluster-2:scan-2": {Scan: &storage.ComplianceOperatorScanV2{Id: "scan-2", ClusterId: "cluster-2"}},
			},
		}
		snapshot := &storage.ComplianceOperatorReportSnapshotV2{
			ReportId: "report-1",
			ReportStatus: &storage.ComplianceOperatorReportStatus{
				ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
			},
		}
		err := impl.generateSingleReportFromWatcherResults(result, snapshot)
		m.Require().NoError(err)
		m.Assert().Equal(storage.ComplianceOperatorReportStatus_PREPARING, snapshot.GetReportStatus().GetRunState())

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			m.FailNow("timeout waiting for ProcessReportRequest")
		}
	})

	m.Run("Report generation with failed clusters", func() {
		setupReportDataMocks()
		m.complianceIntegrationDataStore.EXPECT().
			GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
		m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).
			Times(len(scanConfig.GetClusters())).
			Return(getScans(len(scanConfig.GetProfiles())), nil)
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		done := make(chan struct{})
		m.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).Times(1).
			DoAndReturn(func(_ any) error {
				close(done)
				return nil
			})

		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		result := &watcher.ScanConfigWatcherResults{
			WatcherID:  scanConfig.GetId(),
			ScanConfig: scanConfig,
			Error:      watcher.ErrScanConfigTimeout,
			ScanResults: map[string]*watcher.ScanWatcherResults{
				"cluster-1:scan-1": {
					Scan:  &storage.ComplianceOperatorScanV2{Id: "scan-1", ClusterId: "cluster-1"},
					Error: watcher.ErrScanTimeout,
				},
			},
		}
		snapshot := &storage.ComplianceOperatorReportSnapshotV2{
			ReportId: "report-1",
			ReportStatus: &storage.ComplianceOperatorReportStatus{
				ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
			},
		}
		err := impl.generateSingleReportFromWatcherResults(result, snapshot)
		m.Require().NoError(err)
		m.Assert().Equal(storage.ComplianceOperatorReportStatus_PREPARING, snapshot.GetReportStatus().GetRunState())
		m.Assert().NotEmpty(snapshot.GetReportStatus().GetErrorMsg())

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			m.FailNow("timeout waiting for ProcessReportRequest")
		}
	})
}

func (m *ManagerTestSuite) TestCreateAutomaticSnapshotAndSubscribe() {
	scanConfig := getTestScanConfig()

	m.Run("No notifiers returns error", func() {
		noNotifierConfig := getTestScanConfig()
		noNotifierConfig.Notifiers = nil

		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		stubWatcher := &stubScanConfigWatcher{}
		err := impl.createAutomaticSnapshotAndSubscribe(m.ctx, noNotifierConfig, stubWatcher)
		m.Require().Error(err)
		m.Assert().Len(stubWatcher.subscribedSnapshots, 0)
	})

	m.Run("With notifiers subscribes and upserts snapshot", func() {
		m.scanConfigDataStore.EXPECT().GetScanConfigClusterStatus(gomock.Any(), gomock.Eq(scanConfig.GetId())).
			Times(1).Return(getTestClusterStatusFromScanConfig(scanConfig), nil)
		m.suiteDataStore.EXPECT().GetSuites(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
		m.bindingsDataStore.EXPECT().GetScanSettingBindings(gomock.Any(), gomock.Any()).
			Times(len(scanConfig.GetClusters())).Return(nil, nil)
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).Return(nil)

		mgr := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
		impl := mgr.(*managerImpl)
		stubWatcher := &stubScanConfigWatcher{}
		err := impl.createAutomaticSnapshotAndSubscribe(m.ctx, scanConfig, stubWatcher)
		m.Require().NoError(err)
		m.Assert().Len(stubWatcher.subscribedSnapshots, 1)
		m.Assert().Equal(storage.ComplianceOperatorReportStatus_WAITING, stubWatcher.subscribedSnapshots[0].GetReportStatus().GetRunState())
		m.Assert().Equal(storage.ComplianceOperatorReportStatus_SCHEDULED, stubWatcher.subscribedSnapshots[0].GetReportStatus().GetReportRequestType())
	})
}

func (m *ManagerTestSuite) TestHandleReadyScanDeleteOldResultsGate() {
	now := protocompat.TimestampNow()
	scan := &storage.ComplianceOperatorScanV2{
		Id:              "scan-1",
		ClusterId:       "cluster-1",
		ScanConfigName:  "test-scan",
		ScanRefId:       "ref-1",
		LastStartedTime: now,
	}

	tests := map[string]struct {
		err      error
		expectFn func(done chan struct{}, checkResultDS *checkResultsMocks.MockDataStore, scanConfigDS *scanConfigurationDS.MockDataStore)
	}{
		"success should call DeleteOldResults": {
			err: nil,
			expectFn: func(done chan struct{}, checkResultDS *checkResultsMocks.MockDataStore, scanConfigDS *scanConfigurationDS.MockDataStore) {
				checkResultDS.EXPECT().
					DeleteOldResults(gomock.Any(), gomock.Eq(scan.GetLastStartedTime()), gomock.Eq(scan.GetScanRefId()), gomock.Eq(false)).
					Times(1).Return(nil)
				scanConfigDS.EXPECT().
					GetScanConfigurationByName(gomock.Any(), gomock.Eq(scan.GetScanConfigName())).
					Times(1).
					DoAndReturn(func(_ any, _ any) (*storage.ComplianceOperatorScanConfigurationV2, error) {
						close(done)
						return nil, errors.New("stop here")
					})
			},
		},
		"ErrScanRemoved should call DeleteOldResults with includeCurrentResults": {
			err: watcher.ErrScanRemoved,
			expectFn: func(done chan struct{}, checkResultDS *checkResultsMocks.MockDataStore, scanConfigDS *scanConfigurationDS.MockDataStore) {
				checkResultDS.EXPECT().
					DeleteOldResults(gomock.Any(), gomock.Eq(scan.GetLastStartedTime()), gomock.Eq(scan.GetScanRefId()), gomock.Eq(true)).
					Times(1).
					DoAndReturn(func(_, _, _, _ any) error {
						close(done)
						return nil
					})
			},
		},
		"ErrScanTimeout should NOT call DeleteOldResults": {
			err: watcher.ErrScanTimeout,
			expectFn: func(done chan struct{}, checkResultDS *checkResultsMocks.MockDataStore, scanConfigDS *scanConfigurationDS.MockDataStore) {
				scanConfigDS.EXPECT().
					GetScanConfigurationByName(gomock.Any(), gomock.Eq(scan.GetScanConfigName())).
					Times(1).
					DoAndReturn(func(_ any, _ any) (*storage.ComplianceOperatorScanConfigurationV2, error) {
						close(done)
						return nil, errors.New("stop here")
					})
			},
		},
		"ErrScanContextCancelled should NOT call DeleteOldResults": {
			err: watcher.ErrScanContextCancelled,
			expectFn: func(done chan struct{}, checkResultDS *checkResultsMocks.MockDataStore, scanConfigDS *scanConfigurationDS.MockDataStore) {
				scanConfigDS.EXPECT().
					GetScanConfigurationByName(gomock.Any(), gomock.Eq(scan.GetScanConfigName())).
					Times(1).
					DoAndReturn(func(_ any, _ any) (*storage.ComplianceOperatorScanConfigurationV2, error) {
						close(done)
						return nil, errors.New("stop here")
					})
			},
		},
	}

	for name, tc := range tests {
		m.Run(name, func() {
			ctrl := gomock.NewController(m.T())
			checkResultDS := checkResultsMocks.NewMockDataStore(ctrl)
			scanConfigDS := scanConfigurationDS.NewMockDataStore(ctrl)
			scanDS := scanMocks.NewMockDataStore(ctrl)
			profileDS := profileMocks.NewMockDataStore(ctrl)
			snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
			integrationDS := integrationMocks.NewMockDataStore(ctrl)
			suiteStore := suiteDS.NewMockDataStore(ctrl)
			bindingsStore := bindingsDS.NewMockDataStore(ctrl)
			generator := reportGen.NewMockComplianceReportGenerator(ctrl)

			manager := New(scanConfigDS, scanDS, profileDS, snapshotDS, integrationDS, suiteStore, bindingsStore, checkResultDS, generator)
			manager.Start()
			defer manager.Stop()
			managerImp := manager.(*managerImpl)

			result := &watcher.ScanWatcherResults{
				WatcherID: "watcher-1",
				Scan:      scan,
				Error:     tc.err,
			}

			done := make(chan struct{})
			tc.expectFn(done, checkResultDS, scanConfigDS)

			// Seed a start time entry so the metrics code doesn't panic.
			concurrency.WithLock(&managerImp.watchingScansLock, func() {
				managerImp.watchingScansStartTime[result.WatcherID] = time.Now()
			})

			managerImp.readyQueue.Push(result)

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				m.FailNow("timeout waiting for handleReadyScan to process the result")
			}

			ctrl.Finish()
		})
	}
}

func (m *ManagerTestSuite) TestHandleScan() {
	m.scanConfigDataStore.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
		Return(
			[]*storage.ComplianceOperatorScanConfigurationV2{
				{
					Id: "scan-config-id",
				},
			}, nil,
		)
	m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
		Return([]*storage.ComplianceOperatorReportSnapshotV2{}, nil)
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
	managerImplementation, ok := manager.(*managerImpl)
	require.True(m.T(), ok)
	scan := &storage.ComplianceOperatorScanV2{
		ClusterId: "cluster-id",
	}
	err := manager.HandleScan(context.Background(), scan.CloneVT())
	assert.Error(m.T(), err)

	scan.Id = "scan-id"
	err = manager.HandleScan(context.Background(), scan.CloneVT())
	assert.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		assert.Len(m.T(), managerImplementation.watchingScans, 0)
	})

	scan.LastStartedTime = protocompat.TimestampNow()
	err = manager.HandleScan(context.Background(), scan.CloneVT())
	assert.NoError(m.T(), err)
	id, err := watcher.GetWatcherIDFromScan(context.Background(), scan, m.snapshotDataStore, m.scanConfigDataStore, nil)
	require.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
	})
}

func (m *ManagerTestSuite) TestHandleScanRemove() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
	managerImplementation, ok := manager.(*managerImpl)
	require.True(m.T(), ok)
	manager.Start()
	defer manager.Stop()

	scanID := "scan-1"
	scan := &storage.ComplianceOperatorScanV2{
		Id:              scanID,
		ClusterId:       "cluster-id",
		LastStartedTime: protocompat.TimestampNow(),
	}

	m.Run("GetScan datastore failure", func() {
		m.scanDataStore.EXPECT().GetScan(gomock.Any(), gomock.Any()).Times(1).Return(nil, false, errors.New("some error"))
		err := manager.HandleScanRemove(scanID)
		m.Require().Error(err)
	})

	m.Run("Scan not found", func() {
		m.scanDataStore.EXPECT().GetScan(gomock.Any(), gomock.Any()).Times(1).Return(nil, false, nil)
		err := manager.HandleScanRemove(scanID)
		m.Require().Error(err)
	})

	m.Run("No scan watcher running for scan", func() {
		m.scanDataStore.EXPECT().GetScan(gomock.Any(), gomock.Any()).Times(1).Return(scan, true, nil)
		err := manager.HandleScanRemove(scanID)
		m.Require().NoError(err)
	})

	m.Run("Scan watcher running for scan", func() {
		m.scanConfigDataStore.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
			Return(
				[]*storage.ComplianceOperatorScanConfigurationV2{
					{
						Id: "scan-config-id",
					},
				}, nil,
			)
		waitForDeleteCall := concurrency.NewWaitGroup(1)
		m.checkResultDataStore.EXPECT().DeleteOldResults(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(true)).
			Times(1).DoAndReturn(func(_, _, _, _ any) error {
			waitForDeleteCall.Add(-1)
			return nil
		})
		m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
			Return([]*storage.ComplianceOperatorReportSnapshotV2{}, nil)
		m.scanDataStore.EXPECT().GetScan(gomock.Any(), gomock.Any()).Times(1).Return(scan, true, nil)
		m.Require().NoError(manager.HandleScan(context.Background(), scan))
		id, err := watcher.GetWatcherIDFromScan(context.Background(), scan, m.snapshotDataStore, m.scanConfigDataStore, nil)
		require.NoError(m.T(), err)
		concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
			w, ok := managerImplementation.watchingScans[id]
			assert.True(m.T(), ok)
			assert.NotNil(m.T(), w)
		})
		err = manager.HandleScanRemove(scanID)
		m.Require().NoError(err)
		m.Assert().Eventually(func() bool {
			return concurrency.WithLock1[bool](&managerImplementation.watchingScansLock, func() bool {
				return len(managerImplementation.watchingScans) == 0
			})
		}, 500*time.Millisecond, 10*time.Millisecond)
		select {
		case <-waitForDeleteCall.Done():
		case <-time.After(500 * time.Millisecond):
			m.FailNow("timeout waiting for DeleteOldResults to be called")
		}

	})
}

func (m *ManagerTestSuite) TestHandleResult() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.checkResultDataStore, m.reportGen)
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
		Return([]*storage.ComplianceOperatorScanV2{scan}, nil)
	m.scanConfigDataStore.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
		Return(
			[]*storage.ComplianceOperatorScanConfigurationV2{
				{
					Id: "scan-config-id",
				},
			}, nil,
		)
	m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
		Return([]*storage.ComplianceOperatorReportSnapshotV2{}, nil)
	id, err := watcher.GetWatcherIDFromCheckResult(context.Background(), result, m.scanDataStore, m.snapshotDataStore, m.scanConfigDataStore)
	require.NoError(m.T(), err)
	err = manager.HandleResult(context.Background(), result.CloneVT())
	assert.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
		delete(managerImplementation.watchingScans, id)
	})

	scan.LastStartedTime = timeNowProto
	m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).AnyTimes().
		Return([]*storage.ComplianceOperatorScanV2{scan}, nil)

	err = manager.HandleResult(context.Background(), result.CloneVT())
	assert.Error(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		assert.Len(m.T(), managerImplementation.watchingScans, 0)
	})

	result.Annotations["compliance.openshift.io/last-scanned-timestamp"] = nowRFCFormat
	err = manager.HandleResult(context.Background(), result.CloneVT())
	assert.NoError(m.T(), err)
	id, err = watcher.GetWatcherIDFromCheckResult(context.Background(), result, m.scanDataStore, m.snapshotDataStore, m.scanConfigDataStore)
	require.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
		delete(managerImplementation.watchingScans, id)
	})
	result.Annotations["compliance.openshift.io/last-scanned-timestamp"] = futureRFCFormat
	err = manager.HandleResult(context.Background(), result.CloneVT())
	assert.NoError(m.T(), err)
	id, err = watcher.GetWatcherIDFromCheckResult(context.Background(), result, m.scanDataStore, m.snapshotDataStore, m.scanConfigDataStore)
	require.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
		delete(managerImplementation.watchingScans, id)
	})
}

func getTestScanConfig() *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "test-scan",
		Id:             "test-scan-id",
		Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
			{
				ClusterId: "cluster-1",
			},
			{
				ClusterId: "cluster-2",
			},
		},
		Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
			{
				ProfileName: "profile-1",
			},
			{
				ProfileName: "profile-2",
			},
		},
		Notifiers: []*storage.NotifierConfiguration{
			{
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "notifier-1",
						MailingLists: []string{"test@test.com"},
					},
				},
			},
		},
	}
}

func getTestClusterStatusFromScanConfig(sc *storage.ComplianceOperatorScanConfigurationV2) []*storage.ComplianceOperatorClusterScanConfigStatus {
	ret := make([]*storage.ComplianceOperatorClusterScanConfigStatus, 0, len(sc.GetClusters()))
	for _, c := range sc.GetClusters() {
		ret = append(ret, &storage.ComplianceOperatorClusterScanConfigStatus{
			ClusterId:   c.GetClusterId(),
			ClusterName: fmt.Sprintf("cluster-name-%s", c.GetClusterId()),
		})
	}
	return ret
}

func getScans(numProfiles int) []*storage.ComplianceOperatorScanV2 {
	var ret []*storage.ComplianceOperatorScanV2
	for i := 0; i < numProfiles; i++ {
		name := fmt.Sprintf("profile-%d", i)
		ret = append(ret, &storage.ComplianceOperatorScanV2{
			ScanName: name,
		})
	}
	return ret
}

type stubScanConfigWatcher struct {
	subscribedSnapshots []*storage.ComplianceOperatorReportSnapshotV2
	scans               []*storage.ComplianceOperatorReportSnapshotV2_Scan
}

func (s *stubScanConfigWatcher) PushScanResults(_ *watcher.ScanWatcherResults) error { return nil }

func (s *stubScanConfigWatcher) Subscribe(snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
	s.subscribedSnapshots = append(s.subscribedSnapshots, snapshot)
	return nil
}

func (s *stubScanConfigWatcher) GetScans() []*storage.ComplianceOperatorReportSnapshotV2_Scan {
	return s.scans
}

func (s *stubScanConfigWatcher) Stop() {}

func (s *stubScanConfigWatcher) Finished() concurrency.ReadOnlySignal {
	sig := concurrency.NewSignal()
	return &sig
}
