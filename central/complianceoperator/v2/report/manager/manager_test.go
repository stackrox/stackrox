package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	integrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/complianceReportgenerator/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	m.reportGen = reportGen.NewMockComplianceReportGenerator(m.mockCtrl)
}

func TestComplianceReportManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (m *ManagerTestSuite) TestSubmitReportRequest() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.reportGen)
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
	ctx := context.Background()

	m.Run("Successful report, no watchers running", func() {
		manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.reportGen)
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
		manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.reportGen)
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
		manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.reportGen)
		manager.Start()
		now := protocompat.TimestampNow()
		m.scanConfigDataStore.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
			Return(
				[]*storage.ComplianceOperatorScanConfigurationV2{
					{
						Id: "scan-config-id",
					},
				}, nil,
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
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.reportGen)
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
	id, err := watcher.GetWatcherIDFromScan(context.Background(), scan, m.snapshotDataStore, m.scanConfigDataStore, nil)
	require.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		w, ok := managerImplementation.watchingScans[id]
		assert.True(m.T(), ok)
		assert.NotNil(m.T(), w)
	})
}

func (m *ManagerTestSuite) TestHandleResult() {
	manager := New(m.scanConfigDataStore, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, m.complianceIntegrationDataStore, m.reportGen)
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
		Return([]*storage.ComplianceOperatorScanV2{scan}, nil)

	err = manager.HandleResult(context.Background(), result)
	assert.NoError(m.T(), err)
	concurrency.WithLock(&managerImplementation.watchingScansLock, func() {
		assert.Len(m.T(), managerImplementation.watchingScans, 0)
	})

	result.Annotations["compliance.openshift.io/last-scanned-timestamp"] = nowRFCFormat
	err = manager.HandleResult(context.Background(), result)
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
	err = manager.HandleResult(context.Background(), result)
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
		Return(getTestScanConfig(), nil)
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
