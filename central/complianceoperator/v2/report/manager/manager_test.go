package manager

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	integrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore/mocks"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/utils/strings/slices"
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
	m.reportGen = reportGen.NewMockComplianceReportGenerator(m.mockCtrl)
}

func TestComplianceReportManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (m *ManagerTestSuite) TestSubmitReportRequest() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
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
	identity := mocks.NewMockIdentity(m.mockCtrl)
	identity.EXPECT().UID().AnyTimes().Return("user-id")
	identity.EXPECT().FullName().AnyTimes().Return("user-name")
	identity.EXPECT().FriendlyName().AnyTimes().Return("user-friendly-name")
	m.scanConfigDataStore.EXPECT().GetScanConfigClusterStatus(gomock.Any(), newGetScanConfigClusterStatusMatcher(getTestScanConfig())).AnyTimes().Return(getTestClusterStatusFromScanConfig(getTestScanConfig()), nil)
	m.suiteDataStore.EXPECT().GetSuites(gomock.Any(), newGetSuitesMatcher(getTestScanConfig())).AnyTimes()
	m.bindingsDataStore.EXPECT().GetScanSettingBindings(gomock.Any(), newGetBindingMatcher(getTestScanConfig())).AnyTimes()
	ctx := authn.ContextWithIdentity(context.Background(), identity, m.T())

	m.Run("Successful report, no watchers running", func() {
		manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
		manager.Start()
		wg := concurrency.NewWaitGroup(1)
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(nil)
		m.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).Times(1).
			DoAndReturn(func(_ any) error {
				wg.Add(-1)
				return nil
			})
		err := manager.SubmitReportRequest(ctx, getTestScanConfig(), storage.ComplianceOperatorReportStatus_EMAIL)
		m.Require().NoError(err)
		handleWaitGroup(m.T(), &wg, 10*time.Millisecond, "report generation")
	})

	m.Run("Error in the database", func() {
		manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
		manager.Start()
		wg := concurrency.NewWaitGroup(1)
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_, _ any) error {
				wg.Add(-1)
				return errors.New("some error")
			})
		err := manager.SubmitReportRequest(ctx, getTestScanConfig(), storage.ComplianceOperatorReportStatus_EMAIL)
		m.Require().NoError(err)
		handleWaitGroup(m.T(), &wg, 10*time.Millisecond, "storage error")
	})

	m.Run("Successful report, with watcher running", func() {
		manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
		manager.Start()
		now := protocompat.TimestampNow()
		m.scanConfigDataStore.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
			Return(
				[]*storage.ComplianceOperatorScanConfigurationV2{getTestScanConfig()}, nil,
			)
		m.pushScansAndResults(manager, getTestScanConfig(), now)

		wg := concurrency.NewWaitGroup(2)
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(1).
			Return(nil)
		sc := getTestScanConfig()
		scans := getTestScansFromScanConfig(sc, now)
		scan := getTestScan(scans[0].GetId(), scans[0].GetClusterId(), now, true)
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		m.finishFirstScan(manager, scan, sc)
		m.Eventually(func() bool {
			managerImp, ok := manager.(*managerImpl)
			m.Require().True(ok)
			return concurrency.WithLock1[bool](&managerImp.watchingScanConfigsLock, func() bool {
				return len(managerImp.watchingScanConfigs) > 0
			})
		}, 100*time.Millisecond, 10*time.Millisecond)
		err := manager.SubmitReportRequest(ctx, getTestScanConfig(), storage.ComplianceOperatorReportStatus_EMAIL)
		m.Require().NoError(err)

		time.Sleep(100 * time.Millisecond)

		m.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).Times(2).DoAndReturn(func(_ any) error {
			wg.Add(-1)
			return nil
		})

		m.finishScans(manager, sc, scans[1:])

		m.Eventually(func() bool {
			managerImp, ok := manager.(*managerImpl)
			m.Require().True(ok)
			return concurrency.WithLock1[bool](&managerImp.watchingScanConfigsLock, func() bool {
				return len(managerImp.watchingScanConfigs) == 0
			})
		}, 100*time.Millisecond, 10*time.Millisecond)

		handleWaitGroup(m.T(), &wg, 500*time.Millisecond, "reports to be generated")
	})

}

func (m *ManagerTestSuite) TestFailedReportWithWatcherRunningAndNoNotifiers() {
	m.T().Setenv(env.ReportExecutionMaxConcurrency.EnvVar(), "1")
	identity := mocks.NewMockIdentity(m.mockCtrl)
	identity.EXPECT().UID().AnyTimes().Return("user-id")
	identity.EXPECT().FullName().AnyTimes().Return("user-name")
	identity.EXPECT().FriendlyName().AnyTimes().Return("user-friendly-name")
	m.scanConfigDataStore.EXPECT().GetScanConfigClusterStatus(gomock.Any(), newGetScanConfigClusterStatusMatcher(getTestScanConfig())).AnyTimes().Return(getTestClusterStatusFromScanConfig(getTestScanConfig()), nil)
	m.suiteDataStore.EXPECT().GetSuites(gomock.Any(), newGetSuitesMatcher(getTestScanConfig())).AnyTimes()
	m.bindingsDataStore.EXPECT().GetScanSettingBindings(gomock.Any(), newGetBindingMatcher(getTestScanConfig())).AnyTimes()
	ctx := authn.ContextWithIdentity(context.Background(), identity, m.T())

	// Set the timeouts to 1 and 2 seconds so the scan watchers timeout fast
	m.T().Setenv(env.ComplianceScanWatcherTimeout.EnvVar(), "1s")
	m.T().Setenv(env.ComplianceScanScheduleWatcherTimeout.EnvVar(), "2s")

	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
	manager.Start()
	now := protocompat.TimestampNow()
	scanConfig := getTestScanConfig()
	scanConfig.Notifiers = nil
	m.scanConfigDataStore.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
		Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig}, nil)
	m.pushScansAndResults(manager, scanConfig, now)

	wg := concurrency.NewWaitGroup(1)
	scans := getTestScansFromScanConfig(scanConfig, now)
	scan := getTestScan(scans[0].GetId(), scans[0].GetClusterId(), now, true)
	m.finishFirstScan(manager, scan, scanConfig)

	// The rest of the scan should time out after 1s

	m.scanConfigDataStore.EXPECT().GetScanConfigurationByName(gomock.Any(), gomock.Any()).Times(len(scans)-1).
		Return(scanConfig, nil)
	m.complianceIntegrationDataStore.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return([]*storage.ComplianceIntegration{
			{
				OperatorInstalled: true,
				Version:           "v1.6.0",
			},
		}, nil)
	m.Eventually(func() bool {
		managerImp, ok := manager.(*managerImpl)
		m.Require().True(ok)
		return concurrency.WithLock1[bool](&managerImp.watchingScanConfigsLock, func() bool {
			return len(managerImp.watchingScanConfigs) > 0
		})
	}, 100*time.Millisecond, 10*time.Millisecond)
	gomock.InOrder(
		// The first upsert happens when the download is requested
		// The next upsert is the first scan handling
		// The next three upsert are the next three scans timing out
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).Times(4).Return(nil),
		// We assert that the last upsert has the state as PREPARING
		// We will transition to PARTIAL_ERROR or FAILURE in the reportGen
		m.snapshotDataStore.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).
			Times(1).
			DoAndReturn(func(_ any, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
				assert.Equal(m.T(), storage.ComplianceOperatorReportStatus_PREPARING, snapshot.GetReportStatus().GetRunState())
				wg.Add(-1)
				return nil
			}),
	)
	// We call the reportGen even if all the cluster fail
	m.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).Times(1)
	err := manager.SubmitReportRequest(ctx, scanConfig, storage.ComplianceOperatorReportStatus_DOWNLOAD)
	m.Require().NoError(err)

	managerImp, ok := manager.(*managerImpl)
	m.Require().True(ok)
	m.Eventually(func() bool {
		return concurrency.WithLock1[bool](&managerImp.watchingScanConfigsLock, func() bool {
			return len(managerImp.watchingScanConfigs) == 0
		})
	}, 10*time.Second, 10*time.Millisecond)

	handleWaitGroup(m.T(), &wg, 10*time.Second, "report to fail")

	// The runningReportConfigs should be empty at this point
	m.Eventually(func() bool {
		return concurrency.WithLock1[bool](&managerImp.mu, func() bool {
			return len(managerImp.runningReportConfigs) == 0
		})
	}, 5*time.Second, 100*time.Millisecond)
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
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
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
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
	managerImplementation, ok := manager.(*managerImpl)
	require.True(m.T(), ok)
	manager.Start()

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
	})
}

func (m *ManagerTestSuite) TestHandleResult() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.suiteDataStore, m.bindingsDataStore, m.reportGen)
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
	assert.NoError(m.T(), err)
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

func (m *ManagerTestSuite) pushScansAndResults(manager Manager, sc *storage.ComplianceOperatorScanConfigurationV2, timestamp *protocompat.Timestamp) {
	ctx := context.Background()
	for _, cluster := range sc.GetClusters() {
		for _, profile := range sc.GetProfiles() {
			scan := getTestScan(profile.GetProfileName(), cluster.GetClusterId(), timestamp, false)
			result := getTestResult(scan, timestamp)
			m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(2).
				Return([]*storage.ComplianceOperatorReportSnapshotV2{}, nil)
			m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).
				Return([]*storage.ComplianceOperatorScanV2{scan}, nil)
			err := manager.HandleScan(ctx, scan)
			require.NoError(m.T(), err)
			err = manager.HandleResult(ctx, result)
			require.NoError(m.T(), err)
		}
	}
}

func (m *ManagerTestSuite) finishFirstScan(manager Manager, scan *storage.ComplianceOperatorScanV2, sc *storage.ComplianceOperatorScanConfigurationV2) {
	ctx := context.Background()
	m.profileDataStore.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorProfileV2, error) {
			var ret []*storage.ComplianceOperatorProfileV2
			for range sc.GetClusters() {
				for _, profile := range sc.GetProfiles() {
					ret = append(ret, &storage.ComplianceOperatorProfileV2{
						Name:         profile.GetProfileName(),
						ProfileRefId: profile.GetProfileName(),
					})
				}
			}
			return ret, nil
		})
	idx := 1
	clusterIdx := 1
	m.scanDataStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(len(sc.GetClusters()) * len(sc.GetProfiles())).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
			ret := []*storage.ComplianceOperatorScanV2{
				{
					Id:        fmt.Sprintf("profile-%d", idx),
					ClusterId: fmt.Sprintf("cluster-%d", clusterIdx),
				},
			}
			idx++
			if idx > len(sc.GetProfiles()) {
				idx = 1
				clusterIdx++
			}
			return ret, nil
		})
	m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(2).
		Return([]*storage.ComplianceOperatorReportSnapshotV2{}, nil)
	m.scanConfigDataStore.EXPECT().GetScanConfigurationByName(gomock.Any(), gomock.Any()).Times(1).
		Return(sc, nil)
	err := manager.HandleScan(ctx, scan)
	require.NoError(m.T(), err)
}

func (m *ManagerTestSuite) finishScans(manager Manager, sc *storage.ComplianceOperatorScanConfigurationV2, scans []*storage.ComplianceOperatorScanV2) {
	ctx := context.Background()
	m.scanConfigDataStore.EXPECT().GetScanConfigurationByName(gomock.Any(), gomock.Any()).Times(len(scans)).
		Return(sc, nil)
	m.snapshotDataStore.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(len(scans)).
		Return([]*storage.ComplianceOperatorReportSnapshotV2{}, nil)
	for _, scan := range scans {
		require.NoError(m.T(), manager.HandleScan(ctx, scan))
	}
}

func getTestScansFromScanConfig(sc *storage.ComplianceOperatorScanConfigurationV2, timestamp *protocompat.Timestamp) []*storage.ComplianceOperatorScanV2 {
	var ret []*storage.ComplianceOperatorScanV2
	for _, cluster := range sc.GetClusters() {
		for _, profile := range sc.GetProfiles() {
			ret = append(ret, getTestScan(profile.GetProfileName(), cluster.GetClusterId(), timestamp, true))
		}
	}
	return ret
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

func getTestScan(scan, cluster string, timestamp *timestamppb.Timestamp, done bool) *storage.ComplianceOperatorScanV2 {
	ret := &storage.ComplianceOperatorScanV2{
		Id:              scan,
		ClusterId:       cluster,
		LastStartedTime: timestamp,
	}
	if done {
		ret.Annotations = map[string]string{
			watcher.CheckCountAnnotationKey: "1",
		}
	}
	return ret
}

func getTestResult(scan *storage.ComplianceOperatorScanV2, timestamp *protocompat.Timestamp) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		ScanRefId: scan.GetScanRefId(),
		Annotations: map[string]string{
			watcher.LastScannedAnnotationKey: timestamp.AsTime().Format(time.RFC3339Nano),
		},
	}
}

func handleWaitGroup(t *testing.T, wg *concurrency.WaitGroup, timeout time.Duration, msg string) {
	select {
	case <-time.After(timeout):
		t.Errorf("timeout waiting for %s", msg)
		t.Fail()
	case <-wg.Done():
	}
}

func newGetScanConfigClusterStatusMatcher(sc *storage.ComplianceOperatorScanConfigurationV2) *getScanConfigClusterStatusMatcher {
	return &getScanConfigClusterStatusMatcher{
		scanConfigID: sc.GetId(),
	}
}

type getScanConfigClusterStatusMatcher struct {
	scanConfigID string
	error        string
}

func (m *getScanConfigClusterStatusMatcher) Matches(target interface{}) bool {
	scanConfigID, ok := target.(string)
	if !ok {
		m.error = "target is not of type string"
		return false
	}
	m.error = fmt.Sprintf("expected field scan configuration ID %q", m.scanConfigID)
	return m.scanConfigID == scanConfigID
}

func (m *getScanConfigClusterStatusMatcher) String() string {
	return m.error
}

func newGetSuitesMatcher(sc *storage.ComplianceOperatorScanConfigurationV2) *getSuitesMatcher {
	return &getSuitesMatcher{
		suiteName: sc.GetScanConfigName(),
	}
}

type getSuitesMatcher struct {
	suiteName string
	error     string
}

func (m *getSuitesMatcher) Matches(target interface{}) bool {
	query, ok := target.(*v1.Query)
	if !ok {
		m.error = "target is not of type *v1.Query"
		return false
	}
	m.error = fmt.Sprintf("expected field suite name %q", m.suiteName)
	field := query.GetBaseQuery().GetMatchFieldQuery().GetField()
	if field != search.ComplianceOperatorSuiteName.String() {
		m.error = fmt.Sprintf("unexpected query field %s", field)
		return false
	}
	value := strings.ReplaceAll(query.GetBaseQuery().GetMatchFieldQuery().GetValue(), "\"", "")
	return value == m.suiteName
}

func (m *getSuitesMatcher) String() string {
	return m.error
}

func newGetBindingMatcher(sc *storage.ComplianceOperatorScanConfigurationV2) *getBindingMatcher {
	return &getBindingMatcher{
		scanConfigName: sc.GetScanConfigName(),
		clusters: func() []string {
			ret := make([]string, 0, len(sc.GetClusters()))
			for _, c := range sc.GetClusters() {
				ret = append(ret, c.GetClusterId())
			}
			return ret
		}(),
	}
}

type getBindingMatcher struct {
	scanConfigName string
	clusters       []string
	error          string
}

func (m *getBindingMatcher) Matches(target interface{}) bool {
	query, ok := target.(*v1.Query)
	if !ok {
		m.error = "target is not of type *v1.Query"
		return false
	}
	m.error = fmt.Sprintf("expected fields scan configuration name %q and clusters %v", m.scanConfigName, m.clusters)
	scanConfigFound := false
	clustersFound := false
	for _, q := range query.GetConjunction().GetQueries() {
		field := q.GetBaseQuery().GetMatchFieldQuery().GetField()
		switch field {
		case search.ComplianceOperatorScanConfigName.String():
			value := strings.ReplaceAll(q.GetBaseQuery().GetMatchFieldQuery().GetValue(), "\"", "")
			if value == m.scanConfigName {
				scanConfigFound = true
			}
		case search.ClusterID.String():
			value := strings.ReplaceAll(q.GetBaseQuery().GetMatchFieldQuery().GetValue(), "\"", "")
			if slices.Contains(m.clusters, value) {
				clustersFound = true
			}
		default:
			m.error = fmt.Sprintf("unexpected query field %s", field)
			return false
		}
	}
	return scanConfigFound && clustersFound
}

func (m *getBindingMatcher) String() string {
	return m.error
}
