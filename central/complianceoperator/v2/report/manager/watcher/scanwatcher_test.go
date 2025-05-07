package watcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	coIntegrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testDBAccess = sac.WithAllAccess(context.Background())
)

type testEvent func(*testing.T, ScanWatcher)

func handleScan(id string, startTime *protocompat.Timestamp) func(*testing.T, ScanWatcher) {
	return func(t *testing.T, scanWatcher ScanWatcher) {
		err := scanWatcher.PushScan(&storage.ComplianceOperatorScanV2{
			Id:              id,
			LastStartedTime: startTime,
		})
		require.NoError(t, err)
	}
}

func handleScanWithAnnotation(id, checkCount string, startTime *protocompat.Timestamp) func(*testing.T, ScanWatcher) {
	return func(t *testing.T, scanWatcher ScanWatcher) {
		err := scanWatcher.PushScan(&storage.ComplianceOperatorScanV2{
			Id:              id,
			Annotations:     map[string]string{CheckCountAnnotationKey: checkCount},
			LastStartedTime: startTime,
		})
		require.NoError(t, err)
	}
}

func handleResult(id string, startTime *protocompat.Timestamp) func(*testing.T, ScanWatcher) {
	return func(t *testing.T, scanWatcher ScanWatcher) {
		err := scanWatcher.PushCheckResult(&storage.ComplianceOperatorCheckResultV2{
			CheckId: id,
			Annotations: map[string]string{
				LastScannedAnnotationKey: startTime.AsTime().Format(time.RFC3339Nano),
			},
		})
		require.NoError(t, err)
	}
}

func TestScanWatcher(t *testing.T) {
	timestampNow := protocompat.TimestampNow()
	timeFuture := timestampNow.AsTime().Add(10 * time.Second)
	timestampFuture, err := protocompat.ConvertTimeToTimestampOrError(timeFuture)
	require.NoError(t, err)
	timePast := timestampNow.AsTime().Add(-10 * time.Second)
	timestampPast, err := protocompat.ConvertTimeToTimestampOrError(timePast)
	require.NoError(t, err)

	cases := map[string]struct {
		events          []testEvent
		assertScanID    string
		assertResultIDs []string
	}{
		"scan ready -> result -> result": {
			events: []testEvent{
				handleScanWithAnnotation("id-1", "2", timestampNow),
				handleResult("id-1", timestampNow),
				handleResult("id-2", timestampNow),
			},
			assertScanID:    "id-1",
			assertResultIDs: []string{"id-1", "id-2"},
		},
		"scan -> result -> result -> scan ready": {
			events: []testEvent{
				handleScan("id-1", timestampNow),
				handleResult("id-1", timestampNow),
				handleResult("id-2", timestampNow),
				handleScanWithAnnotation("id-1", "2", timestampNow),
			},
			assertScanID:    "id-1",
			assertResultIDs: []string{"id-1", "id-2"},
		},
		"scan -> result -> scan ready -> result": {
			events: []testEvent{
				handleScan("id-1", timestampNow),
				handleResult("id-1", timestampNow),
				handleScanWithAnnotation("id-1", "2", timestampNow),
				handleResult("id-2", timestampNow),
			},
			assertScanID:    "id-1",
			assertResultIDs: []string{"id-1", "id-2"},
		},
		"result -> result -> scan ready": {
			events: []testEvent{
				handleResult("id-1", timestampNow),
				handleResult("id-2", timestampNow),
				handleScanWithAnnotation("id-1", "2", timestampNow),
			},
			assertScanID:    "id-1",
			assertResultIDs: []string{"id-1", "id-2"},
		},
		"scan -> result -> new scan -> new result -> new result -> scan ready": {
			events: []testEvent{
				handleScan("id-1", timestampNow),
				handleResult("id-1", timestampNow),
				handleScan("id-1", timestampFuture),
				handleResult("id-1", timestampFuture),
				handleResult("id-2", timestampFuture),
				handleScanWithAnnotation("id-1", "2", timestampFuture),
			},
			assertScanID:    "id-1",
			assertResultIDs: []string{"id-1", "id-2"},
		},
		"scan -> result -> new scan -> new result -> new result -> old result -> scan ready": {
			events: []testEvent{
				handleScan("id-1", timestampNow),
				handleResult("id-1", timestampNow),
				handleScan("id-1", timestampFuture),
				handleResult("id-1", timestampFuture),
				handleResult("id-2", timestampFuture),
				handleResult("id-1", timestampPast),
				handleScanWithAnnotation("id-1", "2", timestampFuture),
			},
			assertScanID:    "id-1",
			assertResultIDs: []string{"id-1", "id-2"},
		},
	}
	for tName, tCase := range cases {
		t.Run(tName, func(t *testing.T) {
			watcherID := "id"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resultQueue := queue.NewQueue[*ScanWatcherResults]()
			scanWatcher := NewScanWatcher(ctx, ctx, watcherID, resultQueue)
			for _, event := range tCase.events {
				event(t, scanWatcher)
			}
			require.Eventually(t, func() bool {
				return resultQueue.Len() > 0
			}, 200*time.Millisecond, 10*time.Millisecond)
			result := resultQueue.Pull()
			require.NotNil(t, result)
			assert.Equal(t, tCase.assertScanID, result.Scan.GetId())
			for _, checkID := range tCase.assertResultIDs {
				found := false
				for checkResult := range result.CheckResults {
					if checkID == checkResult {
						found = true
						break
					}
				}
				assert.Truef(t, found, "Expected to find %s", checkID)
			}
		})
	}
}

func TestScanWatcherCancel(t *testing.T) {
	timeNow := protocompat.TimestampNow()
	watcherID := "id"
	ctx, cancel := context.WithCancel(context.Background())
	readyTestQueue := queue.NewQueue[*ScanWatcherResults]()
	scanWatcher := NewScanWatcher(ctx, ctx, watcherID, readyTestQueue)
	handleScan("id-1", timeNow)(t, scanWatcher)
	handleResult("id-1", timeNow)(t, scanWatcher)
	cancel()
	select {
	case <-scanWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	assert.Equal(t, 1, readyTestQueue.Len())
	result := readyTestQueue.Pull()
	assert.ErrorIs(t, result.Error, ErrScanContextCancelled)
}

func TestScanWatcherStop(t *testing.T) {
	timeNow := protocompat.TimestampNow()
	watcherID := "id"
	readyTestQueue := queue.NewQueue[*ScanWatcherResults]()
	scanWatcher := NewScanWatcher(context.Background(), context.Background(), watcherID, readyTestQueue)
	handleScan("id-1", timeNow)(t, scanWatcher)
	handleResult("id-1", timeNow)(t, scanWatcher)
	scanWatcher.Stop(nil)
	select {
	case <-scanWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	assert.Equal(t, 1, readyTestQueue.Len())
	result := readyTestQueue.Pull()
	assert.ErrorIs(t, result.Error, ErrScanContextCancelled)
}

func TestScanWatcherStopWithError(t *testing.T) {
	timeNow := protocompat.TimestampNow()
	watcherID := "id"
	readyTestQueue := queue.NewQueue[*ScanWatcherResults]()
	scanWatcher := NewScanWatcher(context.Background(), context.Background(), watcherID, readyTestQueue)
	handleScan("id-1", timeNow)(t, scanWatcher)
	handleResult("id-1", timeNow)(t, scanWatcher)
	scanWatcher.Stop(ErrScanRemoved)
	select {
	case <-scanWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	assert.Equal(t, 1, readyTestQueue.Len())
	result := readyTestQueue.Pull()
	assert.ErrorIs(t, result.Error, ErrScanRemoved)
}

func TestScanWatcherTimeout(t *testing.T) {
	timeNow := protocompat.TimestampNow()
	readyTestQueue := queue.NewQueue[*ScanWatcherResults]()
	ctx, cancel := context.WithCancel(context.Background())
	finishedSignal := concurrency.NewSignal()
	timeoutC := make(chan time.Time)
	defer close(timeoutC)
	timeout := &testTimer{
		ch: timeoutC,
	}
	scanWatcher := &scanWatcherImpl{
		ctx:        ctx,
		sensorCtx:  ctx,
		cancel:     cancel,
		timeout:    timeout,
		scanC:      make(chan *storage.ComplianceOperatorScanV2),
		resultC:    make(chan *storage.ComplianceOperatorCheckResultV2),
		stopped:    &finishedSignal,
		readyQueue: readyTestQueue,
		scanResults: &ScanWatcherResults{
			SensorCtx:    ctx,
			WatcherID:    "id",
			CheckResults: set.NewStringSet(),
		},
	}
	go scanWatcher.run()
	handleScan("id-1", timeNow)(t, scanWatcher)
	handleResult("id-1", timeNow)(t, scanWatcher)
	// We signal the timeout
	timeoutC <- time.Now()
	select {
	case <-scanWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	// We should have a result in the queue with an error
	require.Equal(t, 1, readyTestQueue.Len())
	result := readyTestQueue.Pull()
	assert.Error(t, result.Error)
}

func TestGetIDFromScan(t *testing.T) {
	ctrl := gomock.NewController(t)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	scanConfigDS := scanConfigMocks.NewMockDataStore(ctrl)
	_, err := GetWatcherIDFromScan(testDBAccess, nil, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)
	scan := &storage.ComplianceOperatorScanV2{}
	_, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)
	scan.ClusterId = "cluster-1"
	_, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)
	scan.Id = "scan-1"
	_, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrComplianceOperatorScanMissingLastStartedFiled, err)
	timeNow := protocompat.TimestampNow()
	scan.LastStartedTime = timeNow
	scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).Times(1).
		Return(nil, errors.New("some error"))
	_, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)

	scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).Times(1).
		Return([]*storage.ComplianceOperatorScanConfigurationV2{}, nil)
	_, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)

	scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
		Return(
			[]*storage.ComplianceOperatorScanConfigurationV2{
				{
					Id: "scan-config-id",
				},
			}, nil,
		)
	snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
			return nil, errors.New("db error")
		})
	_, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)
	snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
			return []*storage.ComplianceOperatorReportSnapshotV2{
				{
					ReportId: "report-1",
				},
			}, nil
		})
	_, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrScanAlreadyHandled, err)
	snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(2).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
			return []*storage.ComplianceOperatorReportSnapshotV2{}, nil
		})
	id, err := GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, nil)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s:%s", scan.ClusterId, scan.Id), id)
	timeNow = protocompat.TimestampNow()
	id, err = GetWatcherIDFromScan(testDBAccess, scan, snapshotDS, scanConfigDS, timeNow)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s:%s", scan.ClusterId, scan.Id), id)
}

func TestGetIDFromResult(t *testing.T) {
	timeNow := protocompat.TimestampNow()
	ctrl := gomock.NewController(t)
	scanDS := mocks.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	scanConfigDS := scanConfigMocks.NewMockDataStore(ctrl)

	snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorReportSnapshotV2, error) {
			return []*storage.ComplianceOperatorReportSnapshotV2{}, nil
		})
	scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
		Return(
			[]*storage.ComplianceOperatorScanConfigurationV2{
				{
					Id: "scan-config-id",
				},
			}, nil,
		)

	_, err := GetWatcherIDFromCheckResult(testDBAccess, nil, scanDS, snapshotDS, scanConfigDS)
	assert.Error(t, err)

	// Error querying the Scan DataStore
	scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
			return nil, errors.New("db error")
		})
	result := &storage.ComplianceOperatorCheckResultV2{}
	_, err = GetWatcherIDFromCheckResult(testDBAccess, result, scanDS, snapshotDS, scanConfigDS)
	assert.Error(t, err)

	// No Scan retrieved
	scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
			return nil, nil
		})
	_, err = GetWatcherIDFromCheckResult(testDBAccess, result, scanDS, snapshotDS, scanConfigDS)
	assert.Error(t, err)

	// Scan retrieved successfully
	scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(5).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
			return []*storage.ComplianceOperatorScanV2{
				{
					ClusterId:       "cluster-1",
					Id:              "scan-1",
					LastStartedTime: timeNow,
				},
			}, nil
		})
	// Empty annotation
	_, err = GetWatcherIDFromCheckResult(testDBAccess, result, scanDS, snapshotDS, scanConfigDS)
	assert.Error(t, err)

	// Invalid format in the annotation
	result.Annotations = map[string]string{
		LastScannedAnnotationKey: protocompat.TimestampNow().String(),
	}
	_, err = GetWatcherIDFromCheckResult(testDBAccess, result, scanDS, snapshotDS, scanConfigDS)
	assert.Error(t, err)

	// The timestamp is in the past
	result.Annotations = map[string]string{
		LastScannedAnnotationKey: timeNow.AsTime().Add(-10 * time.Second).Format(time.RFC3339Nano),
	}
	_, err = GetWatcherIDFromCheckResult(testDBAccess, result, scanDS, snapshotDS, scanConfigDS)
	assert.Error(t, err)
	assert.Error(t, ErrComplianceOperatorReceivedOldCheckResult)

	// The timestamp is in the future
	futureTime := timeNow.AsTime().Add(10 * time.Second)
	result.Annotations = map[string]string{
		LastScannedAnnotationKey: futureTime.Format(time.RFC3339Nano),
	}
	id, err := GetWatcherIDFromCheckResult(testDBAccess, result, scanDS, snapshotDS, scanConfigDS)
	assert.NoError(t, err)
	assert.Equal(t, "cluster-1:scan-1", id)

	// The timestamp is the same
	result.Annotations = map[string]string{
		LastScannedAnnotationKey: timeNow.AsTime().Format(time.RFC3339Nano),
	}
	id, err = GetWatcherIDFromCheckResult(testDBAccess, result, scanDS, snapshotDS, scanConfigDS)
	assert.NoError(t, err)
	assert.Equal(t, "cluster-1:scan-1", id)
}

func TestIsComplianceOperatorHealthy(t *testing.T) {
	clusterID := "cluster-id"
	ctrl := gomock.NewController(t)
	ds := coIntegrationMocks.NewMockDataStore(ctrl)

	// DataStore error
	ds.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceIntegration, error) {
			return []*storage.ComplianceIntegration{}, ErrComplianceOperatorIntegrationDataStore
		})

	_, err := IsComplianceOperatorHealthy(testDBAccess, clusterID, ds)
	assert.Error(t, err)

	// No integrations retrieved
	ds.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceIntegration, error) {
			return []*storage.ComplianceIntegration{}, ErrComplianceOperatorIntegrationZeroIntegrations
		})
	_, err = IsComplianceOperatorHealthy(testDBAccess, clusterID, ds)
	assert.Error(t, err)

	// Compliance Operator not installed
	ds.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceIntegration, error) {
			return []*storage.ComplianceIntegration{
				{
					OperatorInstalled: false,
				},
			}, nil
		})
	_, err = IsComplianceOperatorHealthy(testDBAccess, clusterID, ds)
	assert.Error(t, err)
	assert.Error(t, ErrComplianceOperatorNotInstalled, err)

	// Minimum version error
	ds.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceIntegration, error) {
			return []*storage.ComplianceIntegration{
				{
					OperatorInstalled: true,
					Version:           "v1.5.0",
				},
			}, nil
		})
	_, err = IsComplianceOperatorHealthy(testDBAccess, clusterID, ds)
	assert.Error(t, err)
	assert.Equal(t, ErrComplianceOperatorVersion, err)

	// Compliance Operator is healthy
	ds.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), gomock.Any()).Times(1).
		DoAndReturn(func(_, _ any) ([]*storage.ComplianceIntegration, error) {
			return []*storage.ComplianceIntegration{
				{
					OperatorInstalled: true,
					Version:           "v1.6.0",
				},
			}, nil
		})
	_, err = IsComplianceOperatorHealthy(testDBAccess, clusterID, ds)
	assert.NoError(t, err)
}
