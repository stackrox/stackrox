package pipeline

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/processsignal"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testContainerID  = "test-container-1"
	testDeploymentID = "test-deployment-1"
	testSignalID     = "test-signal-1"
)

func newTestDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	d, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.EnrichedProcessIndicatorLane),
		lane.NewBlockingLane(pubsub.UnenrichedProcessIndicatorLane),
	}))
	require.NoError(t, err)
	t.Cleanup(func() { d.Stop() })
	return d
}

func newTestFileActivity(containerID, signalID, path string) *sensorAPI.FileActivity {
	return &sensorAPI.FileActivity{
		Hostname: "test-host",
		Process: &sensorAPI.ProcessSignal{
			Id:          signalID,
			ContainerId: containerID,
			Name:        "test-process",
		},
		File: &sensorAPI.FileActivity_Open{
			Open: &sensorAPI.FileOpen{
				Activity: &sensorAPI.FileActivityBase{
					Path:     path,
					HostPath: "/host" + path,
				},
			},
		},
	}
}

type testPipeline struct {
	*Pipeline
	detector     *mocks.MockDetector
	dispatcher   common.PubSubDispatcher
	activityChan chan *sensorAPI.FileActivity
}

func newTestPipeline(t *testing.T) *testPipeline {
	t.Helper()
	mockCtrl := gomock.NewController(t)
	mockDetector := mocks.NewMockDetector(mockCtrl)
	clusterStore := clusterentities.NewStore(0, nil, false)
	dispatcher := newTestDispatcher(t)
	activityChan := make(chan *sensorAPI.FileActivity, 10)
	p := NewFileSystemPipeline(mockDetector, clusterStore, activityChan, dispatcher)
	t.Cleanup(func() { p.Stop() })
	return &testPipeline{
		Pipeline:     p,
		detector:     mockDetector,
		dispatcher:   dispatcher,
		activityChan: activityChan,
	}
}

func TestFileSystemPipelinePubSubBufferingAndDrain(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		tp := newTestPipeline(t)

		// Send a file activity — in pub/sub mode with no container metadata,
		// it should be buffered and an unenriched event published.
		fa := newTestFileActivity(testContainerID, testSignalID, "/etc/passwd")

		// The enricher subscriber will receive the unenriched event,
		// but we simulate the enrichment by directly publishing an enriched event.
		// First, send the file activity through the pipeline.
		tp.activityChan <- fa

		// Wait for all goroutines to settle.
		synctest.Wait()

		// Verify the activity was buffered.
		tp.activityMutex.Lock()
		key := cacheKey(testContainerID, testSignalID)
		entry := tp.bufferedActivity[key]
		require.NotNil(t, entry, "file activity should be buffered")
		assert.Len(t, entry.activities, 1)
		tp.activityMutex.Unlock()

		// Now simulate the enriched process indicator arriving via pub/sub.
		enrichedIndicator := &storage.ProcessIndicator{
			Id:           "enriched-indicator-1",
			DeploymentId: testDeploymentID,
			Signal: &storage.ProcessSignal{
				Id:          testSignalID,
				ContainerId: testContainerID,
				Name:        "test-process",
			},
		}

		// Expect the detector to receive the file access with the enriched indicator.
		tp.detector.EXPECT().ProcessFileAccess(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, access *storage.FileAccess) {
				assert.Equal(t, testDeploymentID, access.GetProcess().GetDeploymentId())
				assert.Equal(t, "/etc/passwd", access.GetFile().GetEffectivePath())
				assert.Equal(t, storage.FileAccess_OPEN, access.GetOperation())
			},
		)

		enrichedEvent := processsignal.NewEnrichedProcessIndicatorEvent(context.Background(), enrichedIndicator)
		require.NoError(t, tp.dispatcher.Publish(enrichedEvent))

		synctest.Wait()

		// Verify the buffer was drained.
		tp.activityMutex.Lock()
		assert.Empty(t, tp.bufferedActivity)
		assert.Equal(t, 0, tp.totalBufferedActivity)
		tp.activityMutex.Unlock()
	})
}

func TestFileSystemPipelinePubSubMultipleActivitiesSameProcess(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		tp := newTestPipeline(t)

		// Send multiple file activities for the same process.
		paths := []string{"/etc/passwd", "/etc/shadow", "/etc/hosts"}
		for _, path := range paths {
			tp.activityChan <- newTestFileActivity(testContainerID, testSignalID, path)
		}

		synctest.Wait()

		// Verify all activities are buffered under the same key.
		tp.activityMutex.Lock()
		key := cacheKey(testContainerID, testSignalID)
		entry := tp.bufferedActivity[key]
		require.NotNil(t, entry)
		assert.Len(t, entry.activities, 3)
		tp.activityMutex.Unlock()

		// All three should be dispatched when the enriched indicator arrives.
		var received []*storage.FileAccess
		tp.detector.EXPECT().ProcessFileAccess(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
			func(_ context.Context, access *storage.FileAccess) {
				received = append(received, access)
			},
		)

		enrichedIndicator := &storage.ProcessIndicator{
			Id:           "enriched-indicator-1",
			DeploymentId: testDeploymentID,
			Signal: &storage.ProcessSignal{
				Id:          testSignalID,
				ContainerId: testContainerID,
			},
		}
		require.NoError(t, tp.dispatcher.Publish(processsignal.NewEnrichedProcessIndicatorEvent(context.Background(), enrichedIndicator)))

		synctest.Wait()

		require.Len(t, received, 3)
		expectedPaths := map[string]bool{"/etc/passwd": true, "/etc/shadow": true, "/etc/hosts": true}
		for _, access := range received {
			assert.True(t, expectedPaths[access.GetFile().GetEffectivePath()],
				"unexpected path: %s", access.GetFile().GetEffectivePath())
		}
	})
}

func TestFileSystemPipelineHostProcessBypassesPubSub(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		tp := newTestPipeline(t)

		// Host process (empty container ID) should bypass pub/sub and process directly.
		fa := newTestFileActivity("", testSignalID, "/etc/passwd")

		tp.detector.EXPECT().ProcessFileAccess(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, access *storage.FileAccess) {
				assert.Equal(t, "/etc/passwd", access.GetFile().GetEffectivePath())
			},
		)

		tp.activityChan <- fa

		synctest.Wait()

		// Nothing should be buffered.
		tp.activityMutex.Lock()
		assert.Empty(t, tp.bufferedActivity)
		tp.activityMutex.Unlock()
	})
}

func TestFileSystemPipelineBufferExpiration(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		tp := newTestPipeline(t)

		// Send a file activity to get it buffered.
		tp.activityChan <- newTestFileActivity(testContainerID, testSignalID, "/etc/passwd")

		synctest.Wait()

		tp.activityMutex.Lock()
		require.NotEmpty(t, tp.bufferedActivity)
		tp.activityMutex.Unlock()

		// Advance the fake clock past the TTL and cleanup interval
		// so the cleanup goroutine prunes the expired entry.
		time.Sleep(bufferedActivityTTL + bufferCleanupInterval)
		synctest.Wait()

		// Verify the buffer was cleaned up.
		tp.activityMutex.Lock()
		assert.Empty(t, tp.bufferedActivity)
		assert.Equal(t, 0, tp.totalBufferedActivity)
		tp.activityMutex.Unlock()
	})
}

func TestFileSystemPipelineTranslation(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	defer goleak.AssertNoGoroutineLeaks(t)

	base := func(path string) *sensorAPI.FileActivityBase {
		return &sensorAPI.FileActivityBase{Path: path, HostPath: "/host" + path}
	}

	cases := map[string]struct {
		activity      *sensorAPI.FileActivity
		wantOperation storage.FileAccess_Operation
		wantPath      string
		wantHostPath  string
		// Additional assertions for operation-specific metadata.
		check func(t *testing.T, access *storage.FileAccess)
	}{
		"creation": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File:     &sensorAPI.FileActivity_Creation{Creation: &sensorAPI.FileCreation{Activity: base("/etc/passwd")}},
			},
			wantOperation: storage.FileAccess_CREATE,
			wantPath:      "/etc/passwd",
			wantHostPath:  "/host/etc/passwd",
		},
		"unlink": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File:     &sensorAPI.FileActivity_Unlink{Unlink: &sensorAPI.FileUnlink{Activity: base("/tmp/file")}},
			},
			wantOperation: storage.FileAccess_UNLINK,
			wantPath:      "/tmp/file",
			wantHostPath:  "/host/tmp/file",
		},
		"open": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File:     &sensorAPI.FileActivity_Open{Open: &sensorAPI.FileOpen{Activity: base("/etc/shadow")}},
			},
			wantOperation: storage.FileAccess_OPEN,
			wantPath:      "/etc/shadow",
			wantHostPath:  "/host/etc/shadow",
		},
		"rename": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File: &sensorAPI.FileActivity_Rename{Rename: &sensorAPI.FileRename{
					Old: base("/old/path"),
					New: base("/new/path"),
				}},
			},
			wantOperation: storage.FileAccess_RENAME,
			wantPath:      "/old/path",
			wantHostPath:  "/host/old/path",
			check: func(t *testing.T, access *storage.FileAccess) {
				assert.Equal(t, "/new/path", access.GetMoved().GetEffectivePath())
				assert.Equal(t, "/host/new/path", access.GetMoved().GetActualPath())
			},
		},
		"permission change": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File: &sensorAPI.FileActivity_Permission{Permission: &sensorAPI.FilePermissionChange{
					Activity: base("/etc/sudoers"),
					Mode:     0o644,
				}},
			},
			wantOperation: storage.FileAccess_PERMISSION_CHANGE,
			wantPath:      "/etc/sudoers",
			wantHostPath:  "/host/etc/sudoers",
			check: func(t *testing.T, access *storage.FileAccess) {
				assert.Equal(t, uint32(0o644), access.GetFile().GetMeta().GetMode())
			},
		},
		"ownership change": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File: &sensorAPI.FileActivity_Ownership{Ownership: &sensorAPI.FileOwnershipChange{
					Activity: base("/var/log/syslog"),
					Uid:      1000,
					Gid:      1000,
					Username: "app",
					Group:    "app",
				}},
			},
			wantOperation: storage.FileAccess_OWNERSHIP_CHANGE,
			wantPath:      "/var/log/syslog",
			wantHostPath:  "/host/var/log/syslog",
			check: func(t *testing.T, access *storage.FileAccess) {
				meta := access.GetFile().GetMeta()
				assert.Equal(t, uint32(1000), meta.GetUid())
				assert.Equal(t, uint32(1000), meta.GetGid())
				assert.Equal(t, "app", meta.GetUsername())
				assert.Equal(t, "app", meta.GetGroup())
			},
		},
		"ACL change": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File: &sensorAPI.FileActivity_Acl{Acl: &sensorAPI.FileAclChange{
					Activity: base("/etc/passwd"),
					AclType:  sensorAPI.AclType_ACL_TYPE_ACCESS,
					Entries: []*sensorAPI.AclEntry{
						{Tag: sensorAPI.AclTag_ACL_TAG_USER_OBJ, Perm: 6, Id: 0xFFFFFFFF},
						{Tag: sensorAPI.AclTag_ACL_TAG_USER, Perm: 4, Id: 1000},
						{Tag: sensorAPI.AclTag_ACL_TAG_OTHER, Perm: 0, Id: 0xFFFFFFFF},
					},
				}},
			},
			wantOperation: storage.FileAccess_ACL_CHANGE,
			wantPath:      "/etc/passwd",
			wantHostPath:  "/host/etc/passwd",
			check: func(t *testing.T, access *storage.FileAccess) {
				meta := access.GetFile().GetMeta()
				require.NotNil(t, meta)
				assert.Equal(t, storage.AclType_ACL_TYPE_ACCESS, meta.GetAclType())
				require.Len(t, meta.GetAclEntries(), 3)
				assert.Equal(t, storage.AclTag_ACL_TAG_USER_OBJ, meta.GetAclEntries()[0].GetTag())
				assert.Equal(t, uint32(6), meta.GetAclEntries()[0].GetPerm())
				assert.Equal(t, storage.AclTag_ACL_TAG_USER, meta.GetAclEntries()[1].GetTag())
				assert.Equal(t, uint32(1000), meta.GetAclEntries()[1].GetId())
			},
		},
		"unhandled type returns nil": {
			activity: &sensorAPI.FileActivity{
				Hostname: "test-host",
				Process:  &sensorAPI.ProcessSignal{Id: testSignalID, Name: "test-process"},
				File:     &sensorAPI.FileActivity_Write{Write: &sensorAPI.FileWrite{Activity: base("/tmp/data")}},
			},
			// wantOperation is irrelevant — we expect nil access.
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tp := newTestPipeline(t)

				if tc.wantOperation == 0 && name == "unhandled type returns nil" {
					// Unhandled types should be silently dropped.
					tp.activityChan <- tc.activity
					synctest.Wait()
					// No ProcessFileAccess call expected.
					return
				}

				tp.detector.EXPECT().ProcessFileAccess(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, access *storage.FileAccess) {
						assert.Equal(t, tc.wantOperation, access.GetOperation())
						assert.Equal(t, tc.wantPath, access.GetFile().GetEffectivePath())
						assert.Equal(t, tc.wantHostPath, access.GetFile().GetActualPath())
						if tc.check != nil {
							tc.check(t, access)
						}
					},
				)

				tp.activityChan <- tc.activity
				synctest.Wait()
			})
		})
	}
}

func TestFileSystemPipelineStopWaitsForAllGoroutines(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	defer goleak.AssertNoGoroutineLeaks(t)

	synctest.Test(t, func(t *testing.T) {
		tp := newTestPipeline(t)

		// Stop should return without hanging, proving both goroutines exit.
		stopped := false
		go func() {
			tp.Stop()
			stopped = true
		}()

		synctest.Wait()
		assert.True(t, stopped, "Stop() did not return — goroutine leak suspected")
	})
}
