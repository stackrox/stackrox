package watcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	checkResultsMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type scanConfigTestEvent func(*testing.T, ScanConfigWatcher)

func handleInitialScanResults(id string, scanDS *scanMocks.MockDataStore, profileDS *profileDatastore.MockDataStore, numOfScans int) scanConfigTestEvent {
	return func(t *testing.T, watcher ScanConfigWatcher) {
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorProfileV2, error) {
				copv2 := &storage.ComplianceOperatorProfileV2{}
				copv2.SetId(fmt.Sprintf("profile-%s", id))
				return []*storage.ComplianceOperatorProfileV2{
					copv2,
				}, nil
			})
		scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
				ret := make([]*storage.ComplianceOperatorScanV2, numOfScans)
				for i := 0; i < numOfScans; i++ {
					cosv2 := &storage.ComplianceOperatorScanV2{}
					cosv2.SetId(fmt.Sprintf("scan-%d", i))
					ret[i] = cosv2
				}
				return ret, nil
			})
		cosv2 := &storage.ComplianceOperatorScanV2{}
		cosv2.SetId(id)
		err := watcher.PushScanResults(&ScanWatcherResults{
			Scan: cosv2,
		})
		require.NoError(t, err)
	}
}

func handleScanResults(id string) scanConfigTestEvent {
	return func(t *testing.T, watcher ScanConfigWatcher) {
		cosv2 := &storage.ComplianceOperatorScanV2{}
		cosv2.SetId(id)
		err := watcher.PushScanResults(&ScanWatcherResults{
			Scan: cosv2,
		})
		require.NoError(t, err)
	}
}

func handleScanResultsWithError(id string) scanConfigTestEvent {
	return func(t *testing.T, watcher ScanConfigWatcher) {
		cosv2 := &storage.ComplianceOperatorScanV2{}
		cosv2.SetId(id)
		err := watcher.PushScanResults(&ScanWatcherResults{
			Scan:  cosv2,
			Error: errors.New("some error in the scan"),
		})
		require.NoError(t, err)
	}
}

func TestScanConfigWatcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) (*storage.ComplianceOperatorReportSnapshotV2, bool, error) {
			return &storage.ComplianceOperatorReportSnapshotV2{}, true, nil
		})
	snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) error { return nil })
	cases := map[string]struct {
		events           []scanConfigTestEvent
		snapshotIDs      []string
		assertScanIDs    []string
		assertScanErrors []string
	}{
		"one successful scan": {
			events: []scanConfigTestEvent{
				handleInitialScanResults("scan-0", scanDS, profileDS, 1),
			},
			snapshotIDs:   []string{"snapshot-0"},
			assertScanIDs: []string{"scan-0"},
		},
		"two successful scans": {
			events: []scanConfigTestEvent{
				handleInitialScanResults("scan-0", scanDS, profileDS, 2),
				handleScanResults("scan-1"),
			},
			snapshotIDs:   []string{"snapshot-0"},
			assertScanIDs: []string{"scan-0", "scan-1"},
		},
		"two successful scans with two snapshots": {
			events: []scanConfigTestEvent{
				handleInitialScanResults("scan-0", scanDS, profileDS, 2),
				handleScanResults("scan-1"),
			},
			snapshotIDs:   []string{"snapshot-0", "snapshot-1"},
			assertScanIDs: []string{"scan-0", "scan-1"},
		},
		"one successful scan and one failed scan": {
			events: []scanConfigTestEvent{
				handleInitialScanResults("scan-0", scanDS, profileDS, 2),
				handleScanResultsWithError("scan-1"),
			},
			snapshotIDs:      []string{"snapshot-0"},
			assertScanErrors: []string{"scan-1"},
			assertScanIDs:    []string{"scan-0", "scan-1"},
		},
	}
	for tName, tCase := range cases {
		t.Run(tName, func(t *testing.T) {
			watcherID := "sc-id"
			scanConfig := &storage.ComplianceOperatorScanConfigurationV2{}
			scanConfig.SetId(watcherID)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resultsQueue := queue.NewQueue[*ScanConfigWatcherResults]()
			scanConfigWatcher := NewScanConfigWatcher(ctx, ctx, watcherID, scanConfig, scanDS, profileDS, snapshotDS, resultsQueue)
			for _, id := range tCase.snapshotIDs {
				corsv2 := &storage.ComplianceOperatorReportSnapshotV2{}
				corsv2.SetReportId(id)
				require.NoError(t, scanConfigWatcher.Subscribe(corsv2))
			}
			for _, event := range tCase.events {
				event(t, scanConfigWatcher)
			}
			require.Eventually(t, func() bool {
				return resultsQueue.Len() != 0
			}, 200*time.Millisecond, 10*time.Millisecond)
			result := resultsQueue.Pull()
			require.NotNil(t, result)
			require.Len(t, result.ScanResults, len(tCase.assertScanIDs))
			for _, scanResult := range result.ScanResults {
				assert.Contains(t, tCase.assertScanIDs, scanResult.Scan.GetId())
				if scanResult.Error != nil {
					assert.Contains(t, tCase.assertScanErrors, scanResult.Scan.GetId())
				}
			}
			require.Len(t, result.ReportSnapshot, len(tCase.snapshotIDs))
			for _, id := range tCase.snapshotIDs {
				found := false
				for _, snapshot := range result.ReportSnapshot {
					if snapshot.GetReportId() == id {
						found = true
						break
					}
				}
				assert.Truef(t, found, "the Snapshot with id %s was not found", id)
			}
		})
	}
}

func TestScanConfigWatcherCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) (*storage.ComplianceOperatorReportSnapshotV2, bool, error) {
			return &storage.ComplianceOperatorReportSnapshotV2{}, true, nil
		})
	snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) error { return nil })
	watcherID := "sc-id"
	scanConfig := &storage.ComplianceOperatorScanConfigurationV2{}
	scanConfig.SetId(watcherID)
	ctx, cancel := context.WithCancel(context.Background())
	resultQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	scanConfigWatcher := NewScanConfigWatcher(ctx, ctx, watcherID, scanConfig, scanDS, profileDS, snapshotDS, resultQueue)
	handleInitialScanResults("scan-0", scanDS, profileDS, 2)(t, scanConfigWatcher)
	cancel()
	select {
	case <-scanConfigWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	assert.Equal(t, 1, resultQueue.Len())
	result := resultQueue.Pull()
	assert.ErrorIs(t, result.Error, ErrScanConfigContextCancelled)
}

func TestScanConfigWatcherStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) (*storage.ComplianceOperatorReportSnapshotV2, bool, error) {
			return &storage.ComplianceOperatorReportSnapshotV2{}, true, nil
		})
	snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) error { return nil })
	watcherID := "sc-id"
	scanConfig := &storage.ComplianceOperatorScanConfigurationV2{}
	scanConfig.SetId(watcherID)
	resultQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	scanConfigWatcher := NewScanConfigWatcher(context.Background(), context.Background(), watcherID, scanConfig, scanDS, profileDS, snapshotDS, resultQueue)
	handleInitialScanResults("scan-0", scanDS, profileDS, 2)(t, scanConfigWatcher)
	scanConfigWatcher.Stop()
	select {
	case <-scanConfigWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	assert.Equal(t, 1, resultQueue.Len())
	result := resultQueue.Pull()
	assert.ErrorIs(t, result.Error, ErrScanConfigContextCancelled)
}

type testTimer struct {
	ch chan time.Time
}

func (t *testTimer) Stop() bool {
	return true
}

func (t *testTimer) C() <-chan time.Time {
	return t.ch
}

func (t *testTimer) Reset() {}

func TestScanConfigWatcherTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) (*storage.ComplianceOperatorReportSnapshotV2, bool, error) {
			return &storage.ComplianceOperatorReportSnapshotV2{}, true, nil
		})
	snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) error { return nil })
	resultQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	ctx, cancel := context.WithCancel(context.Background())
	finishedSignal := concurrency.NewSignal()
	timeoutC := make(chan time.Time)
	defer close(timeoutC)
	timeout := &testTimer{
		ch: timeoutC,
	}
	coscv2 := &storage.ComplianceOperatorScanConfigurationV2{}
	coscv2.SetId("id")
	scanConfigWatcher := &scanConfigWatcherImpl{
		ctx:                 ctx,
		sensorCtx:           ctx,
		cancel:              cancel,
		stopped:             &finishedSignal,
		scanDS:              scanDS,
		profileDS:           profileDS,
		snapshotDS:          snapshotDS,
		scanWatcherResoutsC: make(chan *ScanWatcherResults),
		scanConfigResults: &ScanConfigWatcherResults{
			SensorCtx:   ctx,
			WatcherID:   "id",
			ScanConfig:  coscv2,
			ScanResults: make(map[string]*ScanWatcherResults),
		},
		readyQueue:  resultQueue,
		scansToWait: set.NewStringSet(),
	}
	go scanConfigWatcher.run(timeout)
	handleInitialScanResults("scan-0", scanDS, profileDS, 2)(t, scanConfigWatcher)
	timeoutC <- time.Now()
	select {
	case <-scanConfigWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	// We should have a result in the queue with an error
	require.Equal(t, 1, resultQueue.Len())
	result := resultQueue.Pull()
	assert.ErrorIs(t, result.Error, ErrScanConfigTimeout)
}

func TestScanConfigWatcherSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) (*storage.ComplianceOperatorReportSnapshotV2, bool, error) {
			return &storage.ComplianceOperatorReportSnapshotV2{}, true, nil
		})
	snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) error { return nil })
	watcherID := "sc-id"
	scanConfig := &storage.ComplianceOperatorScanConfigurationV2{}
	scanConfig.SetId(watcherID)
	resultsQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	scanIDs := []string{"scan-0", "scan-1", "scan-2"}
	snapshotIDS := []string{"snapshot-0", "snapshot-1"}
	scanConfigWatcher := NewScanConfigWatcher(context.Background(), context.Background(), watcherID, scanConfig, scanDS, profileDS, snapshotDS, resultsQueue)
	corsv2 := &storage.ComplianceOperatorReportSnapshotV2{}
	corsv2.SetReportId(snapshotIDS[0])
	err := scanConfigWatcher.Subscribe(corsv2)
	assert.NoError(t, err)
	handleInitialScanResults(scanIDs[0], scanDS, profileDS, len(scanIDs))(t, scanConfigWatcher)
	handleScanResults(scanIDs[1])(t, scanConfigWatcher)
	corsv2h2 := &storage.ComplianceOperatorReportSnapshotV2{}
	corsv2h2.SetReportId(snapshotIDS[1])
	err = scanConfigWatcher.Subscribe(corsv2h2)
	assert.NoError(t, err)
	handleScanResults(scanIDs[2])(t, scanConfigWatcher)

	require.Eventually(t, func() bool {
		return resultsQueue.Len() != 0
	}, 200*time.Millisecond, 10*time.Millisecond)

	require.Equal(t, 1, resultsQueue.Len())
	result := resultsQueue.Pull()
	require.NotNil(t, result)
	require.Len(t, result.ScanResults, len(scanIDs))
	for _, scanResult := range result.ScanResults {
		assert.Contains(t, scanIDs, scanResult.Scan.GetId())
	}
	require.Len(t, result.ReportSnapshot, len(snapshotIDS))
	for _, id := range snapshotIDS {
		found := false
		for _, snapshot := range result.ReportSnapshot {
			if snapshot.GetReportId() == id {
				found = true
				break
			}
		}
		assert.Truef(t, found, "the Snapshot with id %s was not found", id)
	}
}

func TestScanConfigWatcherGetScans(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)
	snapshotDS.EXPECT().GetSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) (*storage.ComplianceOperatorReportSnapshotV2, bool, error) {
			return &storage.ComplianceOperatorReportSnapshotV2{}, true, nil
		})
	snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_, _ any) error { return nil })
	watcherID := "sc-id"
	scanConfig := &storage.ComplianceOperatorScanConfigurationV2{}
	scanConfig.SetId(watcherID)
	resultsQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	scanConfigWatcher := NewScanConfigWatcher(context.Background(), context.Background(), watcherID, scanConfig, scanDS, profileDS, snapshotDS, resultsQueue)
	scans := scanConfigWatcher.GetScans()
	require.Len(t, scans, 0)

	handleInitialScanResults("scan-0", scanDS, profileDS, 2)(t, scanConfigWatcher)
	require.Eventually(t, func() bool {
		return len(scanConfigWatcher.GetScans()) == 1
	}, 200*time.Millisecond, 10*time.Millisecond)

	handleScanResults("scan-1")(t, scanConfigWatcher)
	require.Eventually(t, func() bool {
		return resultsQueue.Len() != 0
	}, 200*time.Millisecond, 10*time.Millisecond)

	scans = scanConfigWatcher.GetScans()
	require.Len(t, scans, 2)
}

func TestDeleteOldResultsFromMissingScans(t *testing.T) {
	ctrl := gomock.NewController(t)
	checkDS := checkResultsMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	timeNow := timestamppb.Now()
	scanID := "scan-id"
	scanRefID := "ref-id"
	t.Run("nil results should return an error", func(tt *testing.T) {
		assert.Error(tt, DeleteOldResultsFromMissingScans(context.Background(), nil, profileDS, scanDS, checkDS))
	})
	t.Run("error retrieving profiles should return an error", func(tt *testing.T) {
		results := &ScanConfigWatcherResults{
			ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{},
		}
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorProfileV2{}, errors.New("some error"))
		assert.Error(tt, DeleteOldResultsFromMissingScans(context.Background(), results, profileDS, scanDS, checkDS))
	})
	t.Run("error retrieving scans set should return an error", func(tt *testing.T) {
		results := &ScanConfigWatcherResults{
			ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{},
		}
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorProfileV2{
			{},
		}, nil)
		scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorScanV2{}, errors.New("some error"))
		assert.Error(tt, DeleteOldResultsFromMissingScans(context.Background(), results, profileDS, scanDS, checkDS))
	})
	t.Run("error retrieving scan should return an error", func(tt *testing.T) {
		results := &ScanConfigWatcherResults{
			ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{},
		}
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorProfileV2{
			{},
		}, nil)
		cosv2 := &storage.ComplianceOperatorScanV2{}
		cosv2.SetId(scanID)
		scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorScanV2{
			cosv2,
		}, nil)
		scanDS.EXPECT().GetScan(gomock.Any(), gomock.Eq(scanID)).Times(1).Return(&storage.ComplianceOperatorScanV2{}, false, errors.New("some error"))
		assert.Error(tt, DeleteOldResultsFromMissingScans(context.Background(), results, profileDS, scanDS, checkDS))
	})
	t.Run("scan not found should return an error", func(tt *testing.T) {
		results := &ScanConfigWatcherResults{
			ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{},
		}
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorProfileV2{
			{},
		}, nil)
		cosv2 := &storage.ComplianceOperatorScanV2{}
		cosv2.SetId(scanID)
		scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorScanV2{
			cosv2,
		}, nil)
		scanDS.EXPECT().GetScan(gomock.Any(), gomock.Eq(scanID)).Times(1).Return(&storage.ComplianceOperatorScanV2{}, false, nil)
		assert.Error(tt, DeleteOldResultsFromMissingScans(context.Background(), results, profileDS, scanDS, checkDS))
	})
	t.Run("delete old results error should return an error", func(tt *testing.T) {
		results := &ScanConfigWatcherResults{
			ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{},
		}
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorProfileV2{
			{},
		}, nil)
		cosv2 := &storage.ComplianceOperatorScanV2{}
		cosv2.SetId(scanID)
		scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorScanV2{
			cosv2,
		}, nil)
		cosv2h2 := &storage.ComplianceOperatorScanV2{}
		cosv2h2.SetScanRefId(scanRefID)
		cosv2h2.SetLastStartedTime(timeNow)
		scanDS.EXPECT().GetScan(gomock.Any(), gomock.Eq(scanID)).Times(1).Return(cosv2h2, true, nil)
		checkDS.EXPECT().DeleteOldResults(gomock.Any(), gomock.Eq(timeNow), gomock.Eq(scanRefID), gomock.Eq(true)).Times(1).Return(errors.New("some error"))
		assert.Error(tt, DeleteOldResultsFromMissingScans(context.Background(), results, profileDS, scanDS, checkDS))
	})
	t.Run("delete old results", func(tt *testing.T) {
		results := &ScanConfigWatcherResults{
			ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{},
		}
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorProfileV2{
			{},
		}, nil)
		cosv2 := &storage.ComplianceOperatorScanV2{}
		cosv2.SetId(scanID)
		scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.ComplianceOperatorScanV2{
			cosv2,
		}, nil)
		cosv2h2 := &storage.ComplianceOperatorScanV2{}
		cosv2h2.SetScanRefId(scanRefID)
		cosv2h2.SetLastStartedTime(timeNow)
		scanDS.EXPECT().GetScan(gomock.Any(), gomock.Eq(scanID)).Times(1).Return(cosv2h2, true, nil)
		checkDS.EXPECT().DeleteOldResults(gomock.Any(), gomock.Eq(timeNow), gomock.Eq(scanRefID), gomock.Eq(true)).Times(1).Return(nil)
		assert.NoError(tt, DeleteOldResultsFromMissingScans(context.Background(), results, profileDS, scanDS, checkDS))
	})
}
