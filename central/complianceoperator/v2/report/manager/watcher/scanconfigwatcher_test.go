package watcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type scanConfigTestEvent func(*testing.T, ScanConfigWatcher)

func handleInitialScanResults(id string, scanDS *scanMocks.MockDataStore, profileDS *profileDatastore.MockDataStore, numOfScans int) scanConfigTestEvent {
	return func(t *testing.T, watcher ScanConfigWatcher) {
		profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorProfileV2, error) {
				return []*storage.ComplianceOperatorProfileV2{
					{
						Id: fmt.Sprintf("profile-%s", id),
					},
				}, nil
			})
		scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_, _ any) ([]*storage.ComplianceOperatorScanV2, error) {
				ret := make([]*storage.ComplianceOperatorScanV2, numOfScans)
				for i := 0; i < numOfScans; i++ {
					ret[i] = &storage.ComplianceOperatorScanV2{
						Id: fmt.Sprintf("scan-%d", i),
					}
				}
				return ret, nil
			})
		err := watcher.PushScanResults(&ScanWatcherResults{
			Scan: &storage.ComplianceOperatorScanV2{
				Id: id,
			},
		})
		require.NoError(t, err)
	}
}

func handleScanResults(id string) scanConfigTestEvent {
	return func(t *testing.T, watcher ScanConfigWatcher) {
		err := watcher.PushScanResults(&ScanWatcherResults{
			Scan: &storage.ComplianceOperatorScanV2{
				Id: id,
			},
		})
		require.NoError(t, err)
	}
}

func handleScanResultsWithError(id string) scanConfigTestEvent {
	return func(t *testing.T, watcher ScanConfigWatcher) {
		err := watcher.PushScanResults(&ScanWatcherResults{
			Scan: &storage.ComplianceOperatorScanV2{
				Id: id,
			},
			Error: errors.New("some error in the scan"),
		})
		require.NoError(t, err)
	}
}

func TestScanConfigWatcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
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
			scanConfig := &storage.ComplianceOperatorScanConfigurationV2{
				Id: watcherID,
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resultsQueue := queue.NewQueue[*ScanConfigWatcherResults]()
			scanConfigWatcher := NewScanConfigWatcher(ctx, watcherID, scanConfig, scanDS, profileDS, resultsQueue, tCase.snapshotIDs...)
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
			require.Len(t, result.ReportSnapshotIDs, len(tCase.snapshotIDs))
			for _, id := range tCase.snapshotIDs {
				assert.Contains(t, result.ReportSnapshotIDs, id)
			}
		})
	}
}

func TestScanConfigWatcherCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	watcherID := "sc-id"
	scanConfig := &storage.ComplianceOperatorScanConfigurationV2{
		Id: watcherID,
	}
	ctx, cancel := context.WithCancel(context.Background())
	resultQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	scanConfigWatcher := NewScanConfigWatcher(ctx, watcherID, scanConfig, scanDS, profileDS, resultQueue)
	handleInitialScanResults("scan-0", scanDS, profileDS, 2)(t, scanConfigWatcher)
	cancel()
	select {
	case <-scanConfigWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	assert.Equal(t, 0, resultQueue.Len())
}

func TestScanConfigWatcherStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	watcherID := "sc-id"
	scanConfig := &storage.ComplianceOperatorScanConfigurationV2{
		Id: watcherID,
	}
	resultQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	scanConfigWatcher := NewScanConfigWatcher(context.Background(), watcherID, scanConfig, scanDS, profileDS, resultQueue)
	handleInitialScanResults("scan-0", scanDS, profileDS, 2)(t, scanConfigWatcher)
	scanConfigWatcher.Stop()
	select {
	case <-scanConfigWatcher.Finished().Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for the watcher to stop")
	}
	assert.Equal(t, 0, resultQueue.Len())
}

func TestScanConfigWatcherTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	resultQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	ctx, cancel := context.WithCancel(context.Background())
	finishedSignal := concurrency.NewSignal()
	scanConfigWatcher := &scanConfigWatcherImpl{
		ctx:       ctx,
		cancel:    cancel,
		stopped:   &finishedSignal,
		scanDS:    scanDS,
		profileDS: profileDS,
		scanC:     make(chan *ScanWatcherResults),
		scanConfigResults: &ScanConfigWatcherResults{
			WatcherID: "id",
			ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{
				Id: "id",
			},
			ScanResults: make(map[string]*ScanWatcherResults),
		},
		readyQueue:  resultQueue,
		scansToWait: set.NewStringSet(),
	}
	timeoutC := make(chan time.Time)
	go scanConfigWatcher.run(timeoutC)
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
	assert.Error(t, result.Error)
}

func TestScanConfigWatcherSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanDS := scanMocks.NewMockDataStore(ctrl)
	profileDS := profileDatastore.NewMockDataStore(ctrl)
	watcherID := "sc-id"
	scanConfig := &storage.ComplianceOperatorScanConfigurationV2{
		Id: watcherID,
	}
	resultsQueue := queue.NewQueue[*ScanConfigWatcherResults]()
	scanIDs := []string{"scan-0", "scan-1", "scan-2"}
	snapshotIDS := []string{"snapshot-0", "snapshot-1"}
	scanConfigWatcher := NewScanConfigWatcher(context.Background(), watcherID, scanConfig, scanDS, profileDS, resultsQueue, snapshotIDS[0])
	handleInitialScanResults(scanIDs[0], scanDS, profileDS, len(scanIDs))(t, scanConfigWatcher)
	handleScanResults(scanIDs[1])(t, scanConfigWatcher)
	scanConfigWatcher.Subscribe(snapshotIDS[1])
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
	require.Len(t, result.ReportSnapshotIDs, len(snapshotIDS))
	for _, id := range snapshotIDS {
		assert.Contains(t, result.ReportSnapshotIDs, id)
	}
}
